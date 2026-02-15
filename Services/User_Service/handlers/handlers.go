// User_Service/handlers/handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"User_Service/config"
	"User_Service/db"
	"User_Service/models"
	"User_Service/services"

	"github.com/gorilla/mux"
)

// UserHandler обрабатывает запросы пользователей
type UserHandler struct {
	config        *config.Config
	db            *db.Database
	avatarService *services.AvatarService
}

// NewUserHandler создает новый обработчик пользователей
func NewUserHandler(cfg *config.Config, database *db.Database) *UserHandler {
	database.SetConfig(cfg)
	return &UserHandler{
		config:        cfg,
		db:            database,
		avatarService: services.NewAvatarService(cfg),
	}
}

// RegisterRoutes регистрирует маршруты для пользователей
func (h *UserHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/users", h.GetUsers).Methods("GET", "OPTIONS")
	r.HandleFunc("/users/{login}", h.GetUser).Methods("GET", "OPTIONS")
	r.HandleFunc("/users", h.CreateUser).Methods("POST", "OPTIONS")
	r.HandleFunc("/users/{login}", h.UpdateUser).Methods("PUT", "OPTIONS")
	r.HandleFunc("/users/{login}", h.DeleteUser).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/users/{login}/avatar", h.UploadAvatar).Methods("POST", "OPTIONS")
	r.HandleFunc("/users/search", h.SearchUsers).Methods("GET", "OPTIONS")
}

// GetUsers получает список всех пользователей с пагинацией
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Парсинг параметров пагинации
	limit, offset := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, total, err := h.db.GetAllUsers(ctx, limit, offset)
	if err != nil {
		log.Printf("Error getting users: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to get users")
		return
	}

	// Дополнение URL аватаров
	h.avatarService.FillAvatarURLs(users)

	respondWithJSON(w, http.StatusOK, models.UserListResponse{
		Users: users,
		Total: total,
	})
}

// GetUser получает пользователя по логину
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.db.GetUserByLogin(ctx, login)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	// Дополнение URL аватара
	h.avatarService.FillAvatarURL(&user)

	respondWithJSON(w, http.StatusOK, user)
}

// CreateUser создает нового пользователя
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
		return
	}

	// Валидация
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Проверка существования пользователя
	exists, err := h.db.UserExists(ctx, req.Login)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to check user existence")
		return
	}

	if exists {
		respondWithError(w, http.StatusConflict, "user_exists", "User already exists")
		return
	}

	// Создание пользователя
	user := req.ToUser()
	if err := h.db.CreateUser(ctx, user); err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to create user")
		return
	}

	respondWithJSON(w, http.StatusCreated, models.SuccessResponse{
		Status:  "ok",
		Message: "User created successfully",
	})
}

// UpdateUser обновляет данные пользователя
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
		return
	}

	// Валидация
	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Получение текущего пользователя
	currentUser, err := h.db.GetUserByLogin(ctx, login)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	// Обновление только переданных полей
	if req.Name != "" {
		currentUser.Name = req.Name
	}
	if req.Email != "" {
		currentUser.Email = req.Email
	}
	if req.Info != "" {
		currentUser.Info = req.Info
	}
	if req.Picture != "" {
		currentUser.Picture = req.Picture
	}

	// Обновление в БД
	if err := h.db.UpdateUser(ctx, login, currentUser); err != nil {
		log.Printf("Error updating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to update user")
		return
	}

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{
		Status:  "ok",
		Message: "User updated successfully",
	})
}

// DeleteUser удаляет пользователя
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Получение пользователя для удаления аватара
	user, err := h.db.GetUserByLogin(ctx, login)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	// Удаление пользователя из БД
	if err := h.db.DeleteUser(ctx, login); err != nil {
		log.Printf("Error deleting user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to delete user")
		return
	}

	// Удаление аватара (если есть)
	if user.Picture != "" {
		if err := h.avatarService.DeleteAvatar(user.Picture); err != nil {
			log.Printf("Warning: failed to delete avatar: %v", err)
			// Не возвращаем ошибку, т.к. пользователь уже удален
		}
	}

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{
		Status:  "ok",
		Message: "User deleted successfully",
	})
}

// UploadAvatar загружает аватар пользователя
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	login := mux.Vars(r)["login"]

	// Ограничение размера запроса
	r.Body = http.MaxBytesReader(w, r.Body, h.config.Static.MaxUploadSize)

	// Парсинг multipart form
	if err := r.ParseMultipartForm(h.config.Static.MaxUploadSize); err != nil {
		respondWithError(w, http.StatusBadRequest, "file_too_large", "File too large")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "missing_file", "Avatar file required")
		return
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Получение текущего пользователя
	user, err := h.db.GetUserByLogin(ctx, login)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	// Удаление старого аватара
	if user.Picture != "" {
		h.avatarService.DeleteAvatar(user.Picture)
	}

	// Сохранение нового аватара
	filename, err := h.avatarService.SaveAvatar(file, header)
	if err != nil {
		log.Printf("Error saving avatar: %v", err)
		respondWithError(w, http.StatusInternalServerError, "upload_error", err.Error())
		return
	}

	// Обновление пользователя в БД
	user.Picture = filename
	if err := h.db.UpdateUser(ctx, login, user); err != nil {
		log.Printf("Error updating user avatar: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to update avatar")
		return
	}

	// Формирование URL аватара
	h.avatarService.FillAvatarURL(&user)

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{
		Status:  "ok",
		Message: "Avatar uploaded successfully",
		Data: map[string]string{
			"avatar_url": user.Picture,
		},
	})
}

// SearchUsers поиск пользователей
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		respondWithError(w, http.StatusBadRequest, "missing_query", "Search query required")
		return
	}

	limit, offset := h.getPaginationParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, total, err := h.db.SearchUsers(ctx, searchTerm, limit, offset)
	if err != nil {
		log.Printf("Error searching users: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to search users")
		return
	}

	// Дополнение URL аватаров
	h.avatarService.FillAvatarURLs(users)

	respondWithJSON(w, http.StatusOK, models.UserListResponse{
		Users: users,
		Total: total,
	})
}

// getPaginationParams извлекает параметры пагинации из запроса
func (h *UserHandler) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 20 // По умолчанию
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

// --- Вспомогательные функции ---

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, code, message string) {
	respondWithJSON(w, statusCode, models.ErrorResponse{
		Error:   code,
		Message: message,
	})
}
