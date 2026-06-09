// voice_service/sfu/engine.go
//
// Engine — центральная точка SFU:
//   - Управляет комнатами (создание, удаление, поиск)
//   - Создаёт PeerConnection через Pion с нужной конфигурацией
//   - Управляет диапазоном UDP портов для медиапотоков

package sfu

import (
	"fmt"
	"log"
	"sync"
	"time"

	"Voice_Service/config"

	"github.com/google/uuid"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

// Engine — менеджер всего SFU
type Engine struct {
	cfg    *config.Config
	api    *webrtc.API
	rtcCfg webrtc.Configuration

	mu    sync.RWMutex
	rooms map[string]*Room // key: room.ID

	// peerRoom — обратный индекс: peerID → roomID
	// Нужен чтобы быстро найти комнату при дисконнекте
	peerRoom sync.Map
}

// NewEngine — создать Engine, настроить Pion
func NewEngine(cfg *config.Config) *Engine {
	// SettingEngine — низкоуровневые настройки Pion
	s := webrtc.SettingEngine{}

	// Ограничиваем диапазон UDP портов — важно для docker/firewall
	if err := s.SetEphemeralUDPPortRange(cfg.UDPPortMin, cfg.UDPPortMax); err != nil {
		log.Fatalf("[sfu] set UDP port range: %v", err)
	}

	// Включаем поддержку simulcast (для будущего видео)
	// При текущей реализации (только аудио) не влияет на поведение
	s.SetSRTPReplayProtectionWindow(512)

	// MediaEngine — регистрируем поддерживаемые кодеки
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		log.Fatalf("[sfu] register codecs: %v", err)
	}

	// InterceptorRegistry — RTP перехватчики (NACK, RTCP, статистика)
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		log.Fatalf("[sfu] register interceptors: %v", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithSettingEngine(s),
		webrtc.WithMediaEngine(m),
		webrtc.WithInterceptorRegistry(i),
	)

	// ICE серверы
	iceServers := make([]webrtc.ICEServer, 0)

	// STUN
	for _, stun := range cfg.STUNServers {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{stun},
		})
	}

	// TURN (если настроен)
	if cfg.TURNServer != "" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{cfg.TURNServer},
			Username:   cfg.TURNUser,
			Credential: cfg.TURNPass,
		})
	}

	rtcCfg := webrtc.Configuration{
		ICEServers: iceServers,
		// SDPSemantics: Unified Plan — стандарт для multiple tracks
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}

	return &Engine{
		cfg:    cfg,
		api:    api,
		rtcCfg: rtcCfg,
		rooms:  make(map[string]*Room),
	}
}

// CreateRoom — создать новую комнату
func (e *Engine) CreateRoom(name, createdBy string) (*Room, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.rooms) >= e.cfg.MaxRooms {
		return nil, fmt.Errorf("max rooms reached (%d)", e.cfg.MaxRooms)
	}

	id := uuid.NewString()
	room := newRoom(id, name, createdBy)
	e.rooms[id] = room

	log.Printf("[sfu] room created: %s (%s) by %s", id, name, createdBy)
	return room, nil
}

// GetRoom — найти комнату по ID
func (e *Engine) GetRoom(roomID string) (*Room, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.rooms[roomID]
	return r, ok
}

// DeleteRoom — удалить комнату (выгнать всех участников)
func (e *Engine) DeleteRoom(roomID string) error {
	e.mu.Lock()
	room, ok := e.rooms[roomID]
	if !ok {
		e.mu.Unlock()
		return ErrRoomNotFoundErr
	}
	delete(e.rooms, roomID)
	e.mu.Unlock()

	room.Close()
	log.Printf("[sfu] room deleted: %s", roomID)
	return nil
}

// ListRooms — список всех комнат
func (e *Engine) ListRooms() []RoomInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]RoomInfo, 0, len(e.rooms))
	for _, r := range e.rooms {
		result = append(result, r.Info())
	}
	return result
}

// JoinRoom — подключить peer к комнате.
// Создаёт PeerConnection для peer'а и добавляет его в комнату.
// Если комнаты не существует — возвращает ошибку.
// Если peer уже в комнате — возвращает существующий peer.
func (e *Engine) JoinRoom(roomID, peerID, userID, username string) (*Peer, *Room, error) {
	room, ok := e.GetRoom(roomID)
	if !ok {
		return nil, nil, ErrRoomNotFoundErr
	}

	if room.PeerCount() >= e.cfg.MaxPeersInRoom {
		return nil, nil, ErrRoomFullErr
	}

	// Создаём PeerConnection через Pion API
	pc, err := e.api.NewPeerConnection(e.rtcCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create peer connection: %w", err)
	}

	// Добавляем transceiver для получения аудио от клиента
	// Direction: RecvOnly — сервер только получает от этого клиента аудио.
	// Каждый подписчик получит отдельный SendOnly трек.
	if _, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	); err != nil {
		pc.Close()
		return nil, nil, fmt.Errorf("add audio transceiver: %w", err)
	}

	peer := newPeer(peerID, userID, username, roomID, pc)
	go peer.runRenegotiationWorker()
	// Регистрируем обратный индекс
	e.peerRoom.Store(peerID, roomID)

	// Добавляем в комнату (настраивает OnTrack, рассылает уведомления)
	room.AddPeer(peer)

	// Подписываем на уже существующие треки
	//room.subscribeNewPeerToExisting(peer)

	log.Printf("[sfu] peer %s (%s) joined room %s", peerID, username, roomID)
	return peer, room, nil
}

// LeaveRoom — покинуть комнату
func (e *Engine) LeaveRoom(peerID string) {
	roomIDVal, ok := e.peerRoom.LoadAndDelete(peerID)
	if !ok {
		return
	}
	roomID := roomIDVal.(string)

	room, exists := e.GetRoom(roomID)
	if !exists {
		return
	}

	room.RemovePeer(peerID)

	// Автоматически удаляем пустую комнату (опционально)
	if room.IsEmpty() {
		log.Printf("[sfu] room %s is empty, scheduling cleanup", roomID)
		go func() {
			time.Sleep(30 * time.Second)
			if room.IsEmpty() {
				e.mu.Lock()
				delete(e.rooms, roomID)
				e.mu.Unlock()
				log.Printf("[sfu] empty room %s removed", roomID)
			}
		}()
	}
}

// PeerRoom — найти roomID по peerID
func (e *Engine) PeerRoom(peerID string) (string, bool) {
	v, ok := e.peerRoom.Load(peerID)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// Close — завершить работу Engine
func (e *Engine) Close() {
	e.mu.Lock()
	rooms := make([]*Room, 0, len(e.rooms))
	for _, r := range e.rooms {
		rooms = append(rooms, r)
	}
	e.rooms = make(map[string]*Room)
	e.mu.Unlock()

	for _, r := range rooms {
		r.Close()
	}

	log.Println("[sfu] engine closed")
}
