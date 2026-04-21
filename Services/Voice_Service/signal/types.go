// voice_service/signal/types.go
//
// Протокол сигнализации между клиентом и Voice Service.
// Все сообщения идут через WS в JSON.
//
// Клиент → Сервер:
//   join          — войти в комнату
//   leave         — выйти из комнаты
//   offer         — SDP offer от клиента
//   answer        — SDP answer (ответ на re-offer от сервера)
//   ice_candidate — ICE кандидат от клиента
//   mute          — замутить/размутить себя
//   set_layer     — выбрать simulcast слой для входящего потока (будущее)
//
// Сервер → Клиент:
//   room_state    — текущее состояние комнаты (список участников)
//   peer_joined   — новый участник присоединился
//   peer_left     — участник вышел
//   offer         — re-offer от сервера при добавлении нового трека
//   answer        — SDP answer от сервера на offer клиента
//   ice_candidate — ICE кандидат от сервера
//   peer_muted    — изменение mute статуса участника
//   error         — ошибка

package signal

import "encoding/json"

// ── Входящие сообщения (клиент → сервер) ──────────────────────────────────

type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type JoinPayload struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name,omitempty"` // если комната не существует — создаётся
}

type SDPPayload struct {
	SDP string `json:"sdp"`
}

type ICEPayload struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid"`
	SDPMLineIndex uint16 `json:"sdp_mline_index"`
}

type MutePayload struct {
	Muted bool `json:"muted"`
}

// SetLayerPayload — для simulcast: клиент говорит какой слой хочет получать
// от конкретного участника. Пока не реализовано, но тип заложен.
type SetLayerPayload struct {
	PeerID string `json:"peer_id"` // чей поток
	Layer  string `json:"layer"`   // "low" | "medium" | "high"
}

// ── Исходящие сообщения (сервер → клиент) ─────────────────────────────────

type OutgoingMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type RoomStatePayload struct {
	RoomID   string     `json:"room_id"`
	RoomName string     `json:"room_name"`
	Peers    []PeerInfo `json:"peers"`
}

type PeerInfo struct {
	PeerID   string `json:"peer_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Muted    bool   `json:"muted"`
}

type PeerJoinedPayload struct {
	Peer PeerInfo `json:"peer"`
}

type PeerLeftPayload struct {
	PeerID string `json:"peer_id"`
}

type PeerMutedPayload struct {
	PeerID string `json:"peer_id"`
	Muted  bool   `json:"muted"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Константы типов входящих сообщений
const (
	TypeJoin         = "join"
	TypeLeave        = "leave"
	TypeOffer        = "offer"
	TypeAnswer       = "answer"
	TypeICECandidate = "ice_candidate"
	TypeMute         = "mute"
	TypeSetLayer     = "set_layer"
)

// Константы типов исходящих сообщений
const (
	TypeRoomState  = "room_state"
	TypePeerJoined = "peer_joined"
	TypePeerLeft   = "peer_left"
	TypePeerMuted  = "peer_muted"
	TypeError      = "error"
)

// Коды ошибок
const (
	ErrRoomFull       = "room_full"
	ErrRoomNotFound   = "room_not_found"
	ErrAlreadyInRoom  = "already_in_room"
	ErrMaxRooms       = "max_rooms_reached"
	ErrInvalidPayload = "invalid_payload"
	ErrWebRTC         = "webrtc_error"
)
