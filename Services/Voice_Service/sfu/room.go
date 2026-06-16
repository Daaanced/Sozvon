// voice_service/sfu/room.go
//
// Room — голосовая комната.
// Хранит список Peer'ов и управляет подписками:
//   - Когда новый peer присоединяется и начинает публиковать трек,
//     он автоматически рассылается всем существующим участникам.
//   - Когда peer уходит — все отписываются от его трека.

package sfu

import (
	"fmt"
	"log"
	"sync"
	"time"

	"Voice_Service/signal"

	"github.com/pion/webrtc/v3"
)

// Room — голосовая комната
type Room struct {
	ID        string
	Name      string
	CreatedBy string
	CreatedAt time.Time

	mu    sync.RWMutex
	peers map[string]*Peer // key: peer.ID
}

func newRoom(id, name, createdBy string) *Room {
	return &Room{
		ID:        id,
		Name:      name,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		peers:     make(map[string]*Peer),
	}
}

// AddPeer — добавить peer в комнату.
// Настраивает:
//  1. OnTrack — когда peer начинает публиковать аудио, рассылаем трек всем
//  2. Подписку нового peer'а на уже существующие треки
func (r *Room) AddPeer(peer *Peer) {
	r.mu.Lock()
	r.peers[peer.ID] = peer
	existingPeers := r.peersExcept(peer.ID)
	r.mu.Unlock()

	log.Printf("[room %s] peer %s (%s) joined, total peers: %d",
		r.ID, peer.ID, peer.Username, len(existingPeers)+1)

	// Настраиваем обработчик входящего трека от нового peer'а
	peer.pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("[room %s] peer %s publishing track: %s (codec: %s)",
			r.ID, peer.ID, remoteTrack.Kind(), remoteTrack.Codec().MimeType)

		peer.mu.Lock()
		peer.publishTrack = remoteTrack
		peer.mu.Unlock()

		// Рассылаем трек всем остальным участникам комнаты
		r.distributeTrack(peer.ID, remoteTrack)
	})

	// OnConnectionStateChange — cleanup при разрыве
	peer.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[room %s] peer %s connection state: %s", r.ID, peer.ID, state)
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateClosed {
			r.RemovePeer(peer.ID)
		}
	})

	peer.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[room %s] peer %s ICE state: %s", r.ID, peer.ID, state)
	})

	peer.pc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("[room %s] peer %s ICE gathering: %s", r.ID, peer.ID, state)
	})

	peer.pc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Printf("[room %s] peer %s signaling state: %s", r.ID, peer.ID, state)
	})

	// Уведомить всех существующих участников о новом peer'е
	joined := signal.OutgoingMessage{
		Type:    signal.TypePeerJoined,
		Payload: signal.PeerJoinedPayload{Peer: peer.Info()},
	}
	for _, existing := range existingPeers {
		existing.Send(joined)
	}

	// Отправить новому peer'у состояние комнаты
	peer.Send(signal.OutgoingMessage{
		Type:    signal.TypeRoomState,
		Payload: r.stateFor(peer.ID),
	})

	// Подписать нового peer'а на уже активные треки существующих участников
	r.subscribeNewPeerToExisting(peer)
}

// distributeTrack — запускает fan-out горутину для трека издателя.
// Новые подписчики добавляются через addSubscriberToFanout.
func (r *Room) distributeTrack(publisherID string, remoteTrack *webrtc.TrackRemote) {
	// Создаём fan-out менеджер для этого трека
	fanout := newTrackFanout(publisherID)

	// Сохраняем fanout в peer чтобы новые подписчики могли добавиться
	r.mu.RLock()
	publisher, ok := r.peers[publisherID]
	r.mu.RUnlock()
	if ok {
		publisher.mu.Lock()
		publisher.fanout = fanout
		publisher.mu.Unlock()
	}

	// Подписываем всех текущих участников
	r.mu.RLock()
	subscribers := r.peersExcept(publisherID)
	r.mu.RUnlock()

	for _, sub := range subscribers {
		lt, err := sub.SubscribeToTrack(publisherID, remoteTrack)
		if err != nil {
			log.Printf("[room %s] subscribe peer %s to %s: %v", r.ID, sub.ID, publisherID, err)
			continue
		}
		fanout.add(lt)
	}

	// Единственная горутина читает RTP и пишет во все localTracks
	go func() {
		buf := make([]byte, 1500)
		for {
			n, _, err := remoteTrack.Read(buf)
			if err != nil {
				log.Printf("[room %s] track from %s ended: %v", r.ID, publisherID, err)
				return
			}
			fanout.write(buf[:n])
		}
	}()
}

// subscribeNewPeerToExisting — подписать нового peer'а на уже активные треки
// Добавляет localTrack в существующий fanout издателя.
func (r *Room) subscribeNewPeerToExisting(newPeer *Peer) {
	r.mu.RLock()
	existing := r.peersExcept(newPeer.ID)
	r.mu.RUnlock()

	for _, publisher := range existing {
		publisher.mu.RLock()
		remoteTrack := publisher.publishTrack
		fanout := publisher.fanout
		publisher.mu.RUnlock()

		if remoteTrack == nil || fanout == nil {
			continue
		}

		// Проверяем — уже подписан?
		newPeer.mu.RLock()
		_, alreadySubscribed := newPeer.localTracks[publisher.ID]
		newPeer.mu.RUnlock()
		if alreadySubscribed {
			log.Printf("[room] peer %s already subscribed to %s, skipping fanout.add",
				newPeer.ID, publisher.ID)
			continue
		}

		lt, err := newPeer.SubscribeToTrack(publisher.ID, remoteTrack)
		if err != nil {
			log.Printf("[room] new peer %s subscribe to %s: %v",
				newPeer.ID, publisher.ID, err)
			continue
		}

		fanout.add(lt)
		log.Printf("[room] new peer %s subscribed to existing track from %s",
			newPeer.ID, publisher.ID)
	}
}

// RemovePeer — удалить peer из комнаты и отписать всех от его трека
func (r *Room) RemovePeer(peerID string) {
	r.mu.Lock()
	peer, exists := r.peers[peerID]
	if !exists {
		r.mu.Unlock()
		return
	}
	delete(r.peers, peerID)
	remaining := r.peersSnapshot()
	r.mu.Unlock()

	log.Printf("[room %s] peer %s left, remaining: %d", r.ID, peerID, len(remaining))

	// Отписать всех от трека ушедшего peer'а
	for _, p := range remaining {
		p.UnsubscribeFromTrack(peerID)
	}

	// Уведомить оставшихся
	leftMsg := signal.OutgoingMessage{
		Type:    signal.TypePeerLeft,
		Payload: signal.PeerLeftPayload{PeerID: peerID},
	}
	for _, p := range remaining {
		p.Send(leftMsg)
	}

	// Закрыть peer
	peer.Close()
}

// BroadcastMute — разослать изменение mute статуса
func (r *Room) BroadcastMute(senderID string, muted bool) {
	r.mu.RLock()
	peers := r.peersExcept(senderID)
	r.mu.RUnlock()

	msg := signal.OutgoingMessage{
		Type:    signal.TypePeerMuted,
		Payload: signal.PeerMutedPayload{PeerID: senderID, Muted: muted},
	}
	for _, p := range peers {
		p.Send(msg)
	}
}

func (r *Room) BroadcastDeafened(senderID string, deafened bool) {
	r.mu.RLock()
	peers := r.peersExcept(senderID)
	r.mu.RUnlock()

	msg := signal.OutgoingMessage{
		Type: signal.TypePeerDeafened,
		Payload: signal.PeerDeafenedPayload{
			PeerID:   senderID,
			Deafened: deafened,
		},
	}

	for _, p := range peers {
		p.Send(msg)
	}
}

// GetPeer — найти peer по ID
func (r *Room) GetPeer(peerID string) (*Peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.peers[peerID]
	return p, ok
}

// PeerCount — количество участников
func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// IsEmpty — нет ли участников
func (r *Room) IsEmpty() bool {
	return r.PeerCount() == 0
}

// Info — снимок состояния комнаты для REST API
func (r *Room) Info() RoomInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peers := make([]signal.PeerInfo, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p.Info())
	}

	return RoomInfo{
		ID:        r.ID,
		Name:      r.Name,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		PeerCount: len(r.peers),
		Peers:     peers,
	}
}

// stateFor — состояние комнаты для конкретного peer'а (без него самого)
func (r *Room) stateFor(peerID string) signal.RoomStatePayload {
	peers := make([]signal.PeerInfo, 0)
	for id, p := range r.peers {
		if id != peerID {
			peers = append(peers, p.Info())
		}
	}
	return signal.RoomStatePayload{
		RoomID:   r.ID,
		RoomName: r.Name,
		Peers:    peers,
	}
}

// Close — закрыть все PeerConnections в комнате
func (r *Room) Close() {
	r.mu.Lock()
	peers := r.peersSnapshot()
	r.peers = make(map[string]*Peer)
	r.mu.Unlock()

	for _, p := range peers {
		p.Close()
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// peersExcept — список всех peers кроме указанного (без локка, вызывать под r.mu)
func (r *Room) peersExcept(excludeID string) []*Peer {
	result := make([]*Peer, 0, len(r.peers))
	for id, p := range r.peers {
		if id != excludeID {
			result = append(result, p)
		}
	}
	return result
}

// peersSnapshot — все peers (без локка)
func (r *Room) peersSnapshot() []*Peer {
	result := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		result = append(result, p)
	}
	return result
}

// drainTrack — читает и выбрасывает пакеты трека чтобы не забить буфер
func drainTrack(track *webrtc.TrackRemote) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := track.Read(buf); err != nil {
			return
		}
	}
}

// RoomInfo — DTO для REST API
type RoomInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	CreatedBy string            `json:"created_by"`
	CreatedAt time.Time         `json:"created_at"`
	PeerCount int               `json:"peer_count"`
	Peers     []signal.PeerInfo `json:"peers"`
}

// RoomCreateRequest — тело запроса на создание комнаты
type RoomCreateRequest struct {
	Name string `json:"name"`
}

// RoomCreateResponse — ответ на создание комнаты
type RoomCreateResponse struct {
	RoomID string `json:"room_id"`
	Name   string `json:"name"`
}

// ErrRoomFull — комната заполнена
var ErrRoomFullErr = fmt.Errorf("room is full")

// ErrRoomNotFoundErr — комната не найдена
var ErrRoomNotFoundErr = fmt.Errorf("room not found")
