// voice_service/handlers/ws.go
//
// VoiceWSHandler — обработчик WS соединений для сигнализации.
// Gateway проксирует /ws/voice → сюда как /ws.
//
// Жизненный цикл:
//  1. Клиент подключается, токен валидируется из query param
//  2. Создаётся сессия (peerID = UUID, привязан к userID из JWT)
//  3. Клиент шлёт "join" → JoinRoom → SDP negotiation
//  4. Обмен offer/answer/ice
//  5. Медиапоток течёт напрямую по UDP (вне WS)
//  6. При дисконнекте — LeaveRoom

package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"Voice_Service/auth"
	"Voice_Service/config"
	"Voice_Service/sfu"
	"Voice_Service/signal"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	wsWriteWait      = 10 * time.Second
	wsPongWait       = 60 * time.Second
	wsPingPeriod     = (wsPongWait * 9) / 10
	wsMaxMessageSize = 64 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Session — активная WS сессия одного клиента
type Session struct {
	peerID   string
	userID   string
	username string
	conn     *websocket.Conn
	peer     *sfu.Peer // nil до join
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// VoiceWSHandler — обработчик WS для сигнализации
type VoiceWSHandler struct {
	cfg        *config.Config
	engine     *sfu.Engine
	jwtService *auth.JWTService

	sessions sync.Map // key: peerID → *Session
}

func NewVoiceWSHandler(cfg *config.Config, engine *sfu.Engine) *VoiceWSHandler {
	return &VoiceWSHandler{
		cfg:        cfg,
		engine:     engine,
		jwtService: auth.NewJWTService(cfg.JWTSecret),
	}
}

// Handle — точка входа для WS соединения
func (h *VoiceWSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Валидация JWT
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		log.Printf("[ws] invalid token: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(wsMaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		peerID:   uuid.NewString(),
		userID:   claims.UserID,
		username: claims.Username,
		conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
	}

	h.sessions.Store(session.peerID, session)
	log.Printf("[ws] session opened: peer=%s user=%s", session.peerID, session.userID)

	// Запускаем ping
	go h.pingLoop(session)

	// Основной цикл чтения (блокирующий)
	h.readLoop(session)

	// Cleanup
	h.sessions.Delete(session.peerID)
	h.engine.LeaveRoom(session.peerID)
	session.cancel()
	conn.Close()

	log.Printf("[ws] session closed: peer=%s", session.peerID)
}

// readLoop — читает входящие сообщения от клиента
func (h *VoiceWSHandler) readLoop(s *Session) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, data, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("[ws] read error peer=%s: %v", s.peerID, err)
			}
			return
		}

		var msg signal.IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[ws] unmarshal error peer=%s: %v", s.peerID, err)
			continue
		}

		h.dispatch(s, msg)
	}
}

// pingLoop — периодически пингует клиента
func (h *VoiceWSHandler) pingLoop(s *Session) {
	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			s.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			err := s.conn.WriteMessage(websocket.PingMessage, nil)
			s.mu.Unlock()
			if err != nil {
				log.Printf("[ws] ping error peer=%s: %v", s.peerID, err)
				s.cancel()
				return
			}
		}
	}
}

// dispatch — маршрутизация входящих сообщений по типу
func (h *VoiceWSHandler) dispatch(s *Session, msg signal.IncomingMessage) {
	switch msg.Type {
	case signal.TypeJoin:
		h.handleJoin(s, msg.Payload)

	case signal.TypeOffer:
		h.handleOffer(s, msg.Payload)

	case signal.TypeAnswer:
		h.handleAnswer(s, msg.Payload)

	case signal.TypeICECandidate:
		h.handleICE(s, msg.Payload)

	case signal.TypeMute:
		h.handleMute(s, msg.Payload)

	case signal.TypeLeave:
		h.handleLeave(s)

	case signal.TypeSetLayer:
		h.handleSetLayer(s, msg.Payload)

	default:
		log.Printf("[ws] unknown message type: %s peer=%s", msg.Type, s.peerID)
	}
}

// ── Обработчики сообщений ──────────────────────────────────────────────────

func (h *VoiceWSHandler) handleJoin(s *Session, raw json.RawMessage) {
	var payload signal.JoinPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		h.sendError(s, signal.ErrInvalidPayload, "invalid join payload")
		return
	}

	if payload.RoomID == "" {
		h.sendError(s, signal.ErrInvalidPayload, "room_id is required")
		return
	}

	// Если peer уже в комнате — сначала выходим
	s.mu.Lock()
	if s.peer != nil {
		s.mu.Unlock()
		h.engine.LeaveRoom(s.peerID)
		s.mu.Lock()
		s.peer = nil
	}
	s.mu.Unlock()

	peer, _, err := h.engine.JoinRoom(
		payload.RoomID,
		s.peerID,
		s.userID,
		s.username,
	)
	if err != nil {
		switch err {
		case sfu.ErrRoomNotFoundErr:
			h.sendError(s, signal.ErrRoomNotFound, "room not found")
		case sfu.ErrRoomFullErr:
			h.sendError(s, signal.ErrRoomFull, "room is full")
		default:
			h.sendError(s, signal.ErrWebRTC, err.Error())
		}
		return
	}

	s.mu.Lock()
	s.peer = peer
	s.mu.Unlock()

	// Запускаем отдельную горутину для чтения из peer.send и записи в WS
	// (заменяем writeLoop который раньше не имел peer'а)
	go h.peerWriteLoop(s, peer)

	log.Printf("[ws] peer=%s joined room=%s", s.peerID, payload.RoomID)
}

// peerWriteLoop — читает из peer.send и пишет в WS.
// Запускается после join, когда peer создан.
func (h *VoiceWSHandler) peerWriteLoop(s *Session, peer *sfu.Peer) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-peer.SendChan():
			if !ok {
				return
			}
			if err := h.writeJSON(s, msg); err != nil {
				log.Printf("[ws] write error peer=%s: %v", s.peerID, err)
				s.cancel()
				return
			}
		}
	}
}

func (h *VoiceWSHandler) handleOffer(s *Session, raw json.RawMessage) {
	s.mu.Lock()
	peer := s.peer
	s.mu.Unlock()

	if peer == nil {
		h.sendError(s, signal.ErrInvalidPayload, "must join a room first")
		return
	}

	var payload signal.SDPPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		h.sendError(s, signal.ErrInvalidPayload, "invalid offer payload")
		return
	}

	answerSDP, err := peer.HandleOffer(payload.SDP)
	if err != nil {
		log.Printf("[ws] handle offer peer=%s: %v", s.peerID, err)
		h.sendError(s, signal.ErrWebRTC, "failed to process offer")
		return
	}

	// Пустой SDP — offer буферизирован из-за glare, answer придёт позже
	if answerSDP == "" {
		log.Printf("[ws] offer buffered (glare) peer=%s", s.peerID)
		return
	}

	peer.Send(signal.OutgoingMessage{
		Type:    signal.TypeAnswer,
		Payload: signal.SDPPayload{SDP: answerSDP},
	})
}

func (h *VoiceWSHandler) handleAnswer(s *Session, raw json.RawMessage) {
	s.mu.Lock()
	peer := s.peer
	s.mu.Unlock()

	if peer == nil {
		return
	}

	var payload signal.SDPPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		h.sendError(s, signal.ErrInvalidPayload, "invalid answer payload")
		return
	}

	if err := peer.HandleAnswer(payload.SDP); err != nil {
		log.Printf("[ws] handle answer peer=%s: %v", s.peerID, err)
	}
}

func (h *VoiceWSHandler) handleICE(s *Session, raw json.RawMessage) {
	s.mu.Lock()
	peer := s.peer
	s.mu.Unlock()

	if peer == nil {
		return
	}

	var payload signal.ICEPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		h.sendError(s, signal.ErrInvalidPayload, "invalid ice payload")
		return
	}

	if err := peer.AddICECandidate(sfu.ICEFromPayload(payload)); err != nil {
		log.Printf("[ws] add ICE candidate peer=%s: %v", s.peerID, err)
	}
}

func (h *VoiceWSHandler) handleMute(s *Session, raw json.RawMessage) {
	s.mu.Lock()
	peer := s.peer
	s.mu.Unlock()

	if peer == nil {
		return
	}

	var payload signal.MutePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	peer.SetMuted(payload.Muted)

	// Разослать изменение остальным в комнате
	roomID, ok := h.engine.PeerRoom(s.peerID)
	if !ok {
		return
	}
	room, ok := h.engine.GetRoom(roomID)
	if !ok {
		return
	}
	room.BroadcastMute(s.peerID, payload.Muted)
}

func (h *VoiceWSHandler) handleLeave(s *Session) {
	h.engine.LeaveRoom(s.peerID)

	s.mu.Lock()
	s.peer = nil
	s.mu.Unlock()

	log.Printf("[ws] peer=%s left room voluntarily", s.peerID)
}

func (h *VoiceWSHandler) handleSetLayer(s *Session, raw json.RawMessage) {
	s.mu.Lock()
	peer := s.peer
	s.mu.Unlock()

	if peer == nil {
		return
	}

	var payload signal.SetLayerPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	peer.SetPreferredLayer(payload.PeerID, sfu.SimulcastLayer(payload.Layer))
}

// ── helpers ────────────────────────────────────────────────────────────────

func (h *VoiceWSHandler) writeJSON(s *Session, msg signal.OutgoingMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return s.conn.WriteJSON(msg)
}

func (h *VoiceWSHandler) sendError(s *Session, code, message string) {
	h.writeJSON(s, signal.OutgoingMessage{
		Type: signal.TypeError,
		Payload: signal.ErrorPayload{
			Code:    code,
			Message: message,
		},
	})
}
