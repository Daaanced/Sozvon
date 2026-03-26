// Chat_Service\handlers\messages.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Chat_Service/db"
	"Chat_Service/models"

	"github.com/gorilla/mux"
)

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	isMember, err := h.db.IsMember(ctx, chatID, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to check membership")
		return
	}
	if !isMember {
		respondWithError(w, http.StatusForbidden, "forbidden", "You are not a member of this chat")
		return
	}

	msg, err := h.db.SaveMessage(ctx, chatID, userID, req.Text, req.ReplyToID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to save message")
		return
	}

	if activated, _ := h.db.ActivateChat(ctx, chatID); activated {
		members, _ := h.db.GetChatMembers(ctx, chatID)
		h.hub.SendToUsers(members, models.WSMessage{
			Event: "chat:activated",
			Data:  map[string]string{"chatId": chatID},
		})
	}

	members, err := h.db.GetChatMembers(ctx, chatID)
	if err == nil {
		recipients := make([]int, 0, len(members)-1)
		for _, uid := range members {
			if uid != userID {
				recipients = append(recipients, uid)
			}
		}
		h.hub.SendToUsers(recipients, models.WSMessage{
			Event: "message:new",
			Data:  msg,
		})
	}

	respondWithJSON(w, http.StatusCreated, msg)
}

func (h *ChatHandler) ForwardMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var req models.ForwardMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	isMember, err := h.db.IsMember(ctx, req.ToChatID, userID)
	if err != nil || !isMember {
		respondWithError(w, http.StatusForbidden, "forbidden", "Not a member of target chat")
		return
	}

	var sentMessages []*models.Message

	if req.CommentText != "" {
		commentMsg, err := h.db.SaveMessage(ctx, req.ToChatID, userID, req.CommentText, nil)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to save comment")
			return
		}
		sentMessages = append(sentMessages, commentMsg)
	}

	forwarded, err := h.db.SaveForwardedMessages(ctx, req.ToChatID, userID, req.MessageIDs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to forward messages")
		return
	}
	sentMessages = append(sentMessages, forwarded...)

	fwdIDs := make([]string, len(forwarded))
	for i, msg := range forwarded {
		fwdIDs[i] = msg.ID
	}
	fwdAttsMap, _ := h.db.GetForwardedAttachmentsByMessageIDs(ctx, fwdIDs)
	for i, msg := range forwarded {
		if msg.ForwardedFrom == nil {
			continue
		}
		if atts, ok := fwdAttsMap[msg.ID]; ok {
			for j := range atts {
				atts[j].URL = h.config.Media.BaseURL + "/media/" + atts[j].StoreName
			}
			forwarded[i].ForwardedFrom.Attachments = atts
		}
	}

	members, _ := h.db.GetChatMembers(ctx, req.ToChatID)
	recipients := make([]int, 0)
	for _, uid := range members {
		if uid != userID {
			recipients = append(recipients, uid)
		}
	}
	for _, msg := range sentMessages {
		h.hub.SendToUsers(recipients, models.WSMessage{Event: "message:new", Data: msg})
	}

	respondWithJSON(w, http.StatusCreated, sentMessages)
}

func (h *ChatHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	messageID := mux.Vars(r)["messageId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var req models.EditMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "text required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.EditMessage(ctx, messageID, userID, req.Text); err != nil {
		respondWithError(w, http.StatusForbidden, "forbidden", err.Error())
		return
	}

	members, _ := h.db.GetChatMembers(ctx, chatID)
	h.hub.SendToUsers(members, models.WSMessage{
		Event: "message:edited",
		Data:  map[string]string{"chatId": chatID, "messageId": messageID, "text": req.Text},
	})

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{Status: "ok"})
}

func (h *ChatHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	messageID := mux.Vars(r)["messageId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	storeNames, err := h.db.DeleteMessage(ctx, messageID, userID)
	if err != nil {
		respondWithError(w, http.StatusForbidden, "forbidden", err.Error())
		return
	}

	for _, name := range storeNames {
		h.storage.Delete(name)
	}

	members, _ := h.db.GetChatMembers(ctx, chatID)
	h.hub.SendToUsers(members, models.WSMessage{
		Event: "message:deleted",
		Data:  map[string]string{"chatId": chatID, "messageId": messageID},
	})

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{Status: "ok"})
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	if chatID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Chat ID required")
		return
	}

	limit, offset := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	exists, err := h.db.ChatExists(ctx, chatID)
	if err != nil || !exists {
		respondWithError(w, http.StatusNotFound, "chat_not_found", "Chat not found")
		return
	}

	messages, err := h.db.GetChatMessages(ctx, chatID, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get messages")
		return
	}

	h.enrichMessages(ctx, messages)
	respondWithJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetMessagesContext(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	messageID := mux.Vars(r)["messageId"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	messages, err := h.db.GetMessagesAroundID(ctx, chatID, messageID, 25)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get context")
		return
	}

	h.enrichMessages(ctx, messages)
	respondWithJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetUnreadMessages(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := h.db.GetMessagesFromUnread(ctx, chatID, userID, 25)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// nil = нет непрочитанных, клиент сам загрузит обычным способом
	if result == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"messages":      nil,
			"firstUnreadId": nil,
			"totalUnread":   0,
		})
		return
	}

	h.enrichMessages(ctx, result.Messages)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"messages":      result.Messages,
		"firstUnreadId": result.FirstUnreadID,
		"totalUnread":   result.TotalUnread,
		"hasMoreTop":    result.HasMoreTop,
		"hasMoreBottom": result.HasMoreBottom,
	})
}

func (h *ChatHandler) GetMessagesAfter(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	messageID := r.URL.Query().Get("messageId")
	if messageID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "messageId required")
		return
	}

	limit, _ := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	messages, err := h.db.GetMessagesAfterID(ctx, chatID, messageID, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	h.enrichMessages(ctx, messages)
	respondWithJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetMessagesBefore(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]
	messageID := r.URL.Query().Get("messageId")
	if messageID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "messageId required")
		return
	}

	limit, _ := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	messages, err := h.db.GetMessagesBeforeID(ctx, chatID, messageID, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	h.enrichMessages(ctx, messages)
	respondWithJSON(w, http.StatusOK, messages)
}

// enrichMessages подгружает вложения для списка сообщений
func (h *ChatHandler) enrichMessages(ctx context.Context, messages []models.Message) {
	if len(messages) == 0 {
		return
	}

	ids := make([]string, len(messages))
	for i, m := range messages {
		ids[i] = m.ID
	}

	if attachmentsMap, err := h.db.GetAttachmentsByMessageIDs(ctx, ids); err != nil {
		log.Printf("Failed to load attachments: %v", err)
	} else {
		for i, m := range messages {
			if atts, ok := attachmentsMap[m.ID]; ok {
				for j := range atts {
					atts[j].URL = h.config.Media.BaseURL + "/media/" + atts[j].StoreName
				}
				messages[i].Attachments = atts
			}
		}
	}

	if fwdAttsMap, err := h.db.GetForwardedAttachmentsByMessageIDs(ctx, ids); err != nil {
		log.Printf("Failed to load forwarded attachments: %v", err)
	} else {
		for i, m := range messages {
			if m.ForwardedFrom == nil {
				continue
			}
			if atts, ok := fwdAttsMap[m.ID]; ok {
				for j := range atts {
					atts[j].URL = h.config.Media.BaseURL + "/media/" + atts[j].StoreName
				}
				messages[i].ForwardedFrom.Attachments = atts
			}
		}
	}
}

func (h *ChatHandler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	chatID := mux.Vars(r)["chatId"]

	userID, err := h.extractUserIDFromAuth(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.config.Media.MaxFileSize*10)
	if err := r.ParseMultipartForm(h.config.Media.MaxFileSize); err != nil {
		respondWithError(w, http.StatusBadRequest, "file_too_large", "Request too large")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	isMember, err := h.db.IsMember(ctx, chatID, userID)
	if err != nil || !isMember {
		respondWithError(w, http.StatusForbidden, "forbidden", "You are not a member of this chat")
		return
	}

	text := r.FormValue("text")
	replyToIDStr := r.FormValue("replyToId")
	var replyToID *string
	if replyToIDStr != "" {
		replyToID = &replyToIDStr
	}

	metaJSON := r.FormValue("meta")

	var meta []struct {
		Width  *int `json:"width"`
		Height *int `json:"height"`
	}

	if metaJSON != "" {
		_ = json.Unmarshal([]byte(metaJSON), &meta)
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 && text == "" {
		respondWithError(w, http.StatusBadRequest, "empty_message", "Text or files required")
		return
	}

	for _, fh := range files {
		if fh.Size > h.config.Media.MaxFileSize {
			respondWithError(w, http.StatusBadRequest, "file_too_large",
				fmt.Sprintf("File %s exceeds 30MB limit", fh.Filename))
			return
		}
	}

	msg, err := h.db.SaveMessage(ctx, chatID, userID, text, replyToID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to save message")
		return
	}

	var attachments []models.Attachment
	for i, fh := range files {
		file, err := fh.Open()
		if err != nil {
			log.Printf("Failed to open uploaded file: %v", err)
			continue
		}
		var width *int
		var height *int

		if i < len(meta) {
			width = meta[i].Width
			height = meta[i].Height
		}

		saved, err := h.storage.Save(file, fh)
		file.Close()
		if err != nil {
			log.Printf("Failed to save file %s: %v", fh.Filename, err)
			continue
		}

		a := models.Attachment{
			ID:        db.NewAttachmentID(),
			MessageID: msg.ID,
			FileName:  saved.FileName,
			StoreName: saved.ID + getExt(saved.FileName),
			MimeType:  saved.MimeType,
			Size:      saved.Size,
			URL:       saved.URL,
			Width:     width,
			Height:    height,
		}
		if err := h.db.SaveAttachment(ctx, a); err != nil {
			log.Printf("Failed to save attachment record: %v", err)
			h.storage.Delete(a.StoreName)
			continue
		}
		attachments = append(attachments, a)
	}
	msg.Attachments = attachments

	if activated, _ := h.db.ActivateChat(ctx, chatID); activated {
		members, _ := h.db.GetChatMembers(ctx, chatID)
		h.hub.SendToUsers(members, models.WSMessage{
			Event: "chat:activated",
			Data:  map[string]string{"chatId": chatID},
		})
	}

	members, err := h.db.GetChatMembers(ctx, chatID)
	if err == nil {
		recipients := make([]int, 0, len(members))
		for _, uid := range members {
			if uid != userID {
				recipients = append(recipients, uid)
			}
		}
		h.hub.SendToUsers(recipients, models.WSMessage{
			Event: "message:new",
			Data:  msg,
		})
	}

	respondWithJSON(w, http.StatusCreated, msg)
}

func (h *ChatHandler) ServeMedia(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		respondWithError(w, http.StatusBadRequest, "invalid_filename", "Invalid filename")
		return
	}

	filePath := h.storage.FilePath(filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		respondWithError(w, http.StatusNotFound, "not_found", "File not found")
		return
	}

	http.ServeFile(w, r, filePath)
}

func getExt(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return ext
}
