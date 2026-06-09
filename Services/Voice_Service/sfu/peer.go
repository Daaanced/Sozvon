// voice_service/sfu/peer.go
//
// Peer представляет одного участника в комнате.
// Каждый Peer имеет:
//   - PeerConnection (Pion) — WebRTC соединение с клиентом
//   - publishTrack        — входящий аудиотрек от клиента (он публикует)
//   - subscribedTracks    — исходящие треки других участников (он подписан)
//   - send channel        — для отправки сигнальных сообщений через WS

package sfu

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"Voice_Service/signal"

	"github.com/pion/webrtc/v3"
)

// SimulcastLayer — слои simulcast (для будущего видео/демо экрана)
type SimulcastLayer string

const (
	LayerLow    SimulcastLayer = "low"
	LayerMedium SimulcastLayer = "medium"
	LayerHigh   SimulcastLayer = "high"
)

// Peer — один участник голосовой комнаты
type Peer struct {
	ID       string
	UserID   string
	Username string
	RoomID   string

	pc   *webrtc.PeerConnection
	send chan signal.OutgoingMessage // WS handler читает из этого канала

	mu    sync.RWMutex
	muted bool

	// publishTrack — трек который этот peer шлёт на сервер (входящий для нас)
	// Nil до тех пор пока клиент не начнёт публиковать аудио
	publishTrack *webrtc.TrackRemote

	// localTracks — треки созданные для пересылки другим пирам
	// key: peerID источника
	localTracks map[string]*webrtc.TrackLocalStaticRTP

	// subscribedLayers — выбранный simulcast слой для каждого издателя
	// Заложено архитектурно, используется в будущем для видео
	subscribedLayers map[string]SimulcastLayer
	fanout           *trackFanout
	// pendingICE — ICE кандидаты полученные до setRemoteDescription
	pendingICE         []webrtc.ICECandidateInit
	pendingRemoteOffer *webrtc.SessionDescription

	closed bool

	closeOnce sync.Once
	renegCh   chan struct{}
}

func newPeer(id, userID, username, roomID string, pc *webrtc.PeerConnection) *Peer {
	return &Peer{
		ID:               id,
		UserID:           userID,
		Username:         username,
		RoomID:           roomID,
		pc:               pc,
		send:             make(chan signal.OutgoingMessage, 64),
		localTracks:      make(map[string]*webrtc.TrackLocalStaticRTP),
		subscribedLayers: make(map[string]SimulcastLayer),
		pendingICE:       make([]webrtc.ICECandidateInit, 0),
		renegCh:          make(chan struct{}, 1),
	}
}

func (p *Peer) scheduleRenegotiate() {
	log.Printf("[peer %s] scheduleRenegotiate called, signaling state: %s",
		p.ID, p.pc.SignalingState())
	select {
	case p.renegCh <- struct{}{}:
		log.Printf("[peer %s] renegotiation scheduled", p.ID)
	default:
		log.Printf("[peer %s] renegotiation already pending, skipped", p.ID)
	}
}

func (p *Peer) runRenegotiationWorker() {
	for range p.renegCh {
		state := p.pc.SignalingState()
		log.Printf("[peer %s] renegWorker tick, state=%s", p.ID, state)

		if state != webrtc.SignalingStateStable {
			log.Printf("[peer %s] not stable, retrying in 200ms", p.ID)
			time.AfterFunc(200*time.Millisecond, p.scheduleRenegotiate)
			continue
		}
		p.renegotiate()
	}
}

// Send — неблокирующая отправка сообщения в WS
func (p *Peer) Send(msg signal.OutgoingMessage) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if closed {
		return
	}
	// recover на случай гонки между проверкой и send
	defer func() { recover() }()
	select {
	case p.send <- msg:
	default:
		log.Printf("[peer %s] send channel full", p.ID)
	}
}

// SendError — отправить сообщение об ошибке
func (p *Peer) SendError(code, message string) {
	p.Send(signal.OutgoingMessage{
		Type: signal.TypeError,
		Payload: signal.ErrorPayload{
			Code:    code,
			Message: message,
		},
	})
}

// SetMuted — обновить mute статус
func (p *Peer) SetMuted(muted bool) {
	p.mu.Lock()
	p.muted = muted
	p.mu.Unlock()
}

// Muted — получить mute статус
func (p *Peer) Muted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.muted
}

// Info — снимок состояния для отправки клиентам
func (p *Peer) Info() signal.PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return signal.PeerInfo{
		PeerID:   p.ID,
		UserID:   p.UserID,
		Username: p.Username,
		Muted:    p.muted,
	}
}

// HandleOffer — обрабатывает SDP offer от клиента, возвращает answer
// HandleOffer — если сервер уже в have-local-offer, буферизируем
func (p *Peer) HandleOffer(sdp string) (string, error) {
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	state := p.pc.SignalingState()
	log.Printf("[peer %s] HandleOffer called, state=%s", p.ID, state)
	if state != webrtc.SignalingStateStable {
		// Не пытаемся rollback — просто буферизируем
		p.mu.Lock()
		p.pendingRemoteOffer = &offer
		p.mu.Unlock()
		log.Printf("[peer %s] offer buffered, state=%s", p.ID, state)
		return "", nil
	}

	return p.applyOffer(offer)
}

// applyOffer — применить offer и вернуть answer SDP
func (p *Peer) applyOffer(offer webrtc.SessionDescription) (string, error) {
	if err := p.pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("set remote description: %w", err)
	}

	p.mu.Lock()
	pending := p.pendingICE
	p.pendingICE = nil
	p.mu.Unlock()

	for _, c := range pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			log.Printf("[peer %s] add pending ICE: %v", p.ID, err)
		}
	}

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("create answer: %w", err)
	}

	gatherDone := webrtc.GatheringCompletePromise(p.pc)

	if err := p.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("set local description: %w", err)
	}

	<-gatherDone
	return p.pc.LocalDescription().SDP, nil
}

// HandleAnswer — обрабатывает SDP answer от клиента (ответ на re-offer сервера)
func (p *Peer) HandleAnswer(sdp string) error {
	log.Printf("[peer %s] HandleAnswer called, state=%s", p.ID, p.pc.SignalingState())
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}
	if err := p.pc.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("set remote description: %w", err)
	}

	p.mu.Lock()
	pending := p.pendingICE
	p.pendingICE = nil
	pendingOffer := p.pendingRemoteOffer
	p.pendingRemoteOffer = nil
	p.mu.Unlock()

	log.Printf("[peer %s] answer applied, new state=%s, pendingOffer=%v",
		p.ID, p.pc.SignalingState(), pendingOffer != nil)
	for _, c := range pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			log.Printf("[peer %s] add pending ICE: %v", p.ID, err)
		}
	}

	// Применяем буферизированный offer от клиента
	if pendingOffer != nil {
		log.Printf("[peer %s] applying buffered remote offer after answer", p.ID)
		go func() {
			answerSDP, err := p.applyOffer(*pendingOffer)
			if err != nil {
				log.Printf("[peer %s] apply buffered offer: %v", p.ID, err)
				return
			}
			if answerSDP != "" {
				p.Send(signal.OutgoingMessage{
					Type:    signal.TypeAnswer,
					Payload: signal.SDPPayload{SDP: answerSDP},
				})
			}
		}()
	}

	return nil
}

// AddICECandidate — добавить ICE кандидат.
// Если RemoteDescription ещё не установлен — кандидат буферизируется.
func (p *Peer) AddICECandidate(init webrtc.ICECandidateInit) error {
	if p.pc.RemoteDescription() == nil {
		log.Printf("[peer %s] ICE buffered (no remote desc yet): %s",
			p.ID, init.Candidate)
		p.mu.Lock()
		p.pendingICE = append(p.pendingICE, init)
		p.mu.Unlock()
		return nil
	}
	log.Printf("[peer %s] ICE candidate added: %s", p.ID, init.Candidate)
	return p.pc.AddICECandidate(init)
}

// SubscribeToTrack — добавить входящий трек другого пира как исходящий для этого пира.
// Сервер создаёт локальный трек и шлёт re-offer клиенту.
// Возвращает localTrack для записи RTP пакетов.
func (p *Peer) SubscribeToTrack(publisherID string, remoteTrack *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.localTracks[publisherID]; exists {
		// Уже подписан
		return p.localTracks[publisherID], nil
	}

	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		remoteTrack.Codec().RTPCodecCapability,
		remoteTrack.ID(),
		fmt.Sprintf("voice-%s", publisherID),
	)
	if err != nil {
		return nil, fmt.Errorf("create local track: %w", err)
	}

	if _, err := p.pc.AddTrack(localTrack); err != nil {
		return nil, fmt.Errorf("add track to peer connection: %w", err)
	}

	p.localTracks[publisherID] = localTrack

	// Re-offer — сообщаем клиенту о новом треке
	go p.scheduleRenegotiate()

	return localTrack, nil
}

// UnsubscribeFromTrack — удалить трек ушедшего пира.
func (p *Peer) UnsubscribeFromTrack(publisherID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.localTracks[publisherID]; !exists {
		return
	}

	// Удаляем sender из PeerConnection
	for _, sender := range p.pc.GetSenders() {
		track := sender.Track()
		if track != nil && track.StreamID() == fmt.Sprintf("voice-%s", publisherID) {
			if err := p.pc.RemoveTrack(sender); err != nil {
				log.Printf("[peer %s] remove track: %v", p.ID, err)
			}
			break
		}
	}

	delete(p.localTracks, publisherID)
	p.scheduleRenegotiate()
}

// SetPreferredLayer — установить предпочтительный simulcast слой.
// Архитектурная заглушка — активируется когда добавляется видео.
func (p *Peer) SetPreferredLayer(publisherID string, layer SimulcastLayer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribedLayers[publisherID] = layer
	log.Printf("[peer %s] set preferred layer for %s: %s", p.ID, publisherID, layer)
}

// renegotiate — отправить клиенту новый offer после изменения треков.
// Вызывается в горутине.
func (p *Peer) renegotiate() {
	log.Printf("[peer %s] starting renegotiation", p.ID)

	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		log.Printf("[peer %s] create re-offer: %v", p.ID, err)
		return
	}

	gatherDone := webrtc.GatheringCompletePromise(p.pc)

	if err := p.pc.SetLocalDescription(offer); err != nil {
		log.Printf("[peer %s] set local description re-offer: %v", p.ID, err)
		return
	}

	<-gatherDone
	log.Printf("[peer %s] renegotiation offer sent", p.ID)

	p.Send(signal.OutgoingMessage{
		Type:    signal.TypeOffer,
		Payload: signal.SDPPayload{SDP: p.pc.LocalDescription().SDP},
	})
}

// Close — закрыть PeerConnection и канал
func (p *Peer) Close() {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()

		if err := p.pc.Close(); err != nil {
			log.Printf("[peer %s] close pc: %v", p.ID, err)
		}
		close(p.send)
	})
}

// ICECandidateInit — конвертер для JSON payload от клиента
func ICEFromPayload(p signal.ICEPayload) webrtc.ICECandidateInit {
	return webrtc.ICECandidateInit{
		Candidate:     p.Candidate,
		SDPMid:        &p.SDPMid,
		SDPMLineIndex: &p.SDPMLineIndex,
	}
}

// MarshalJSON helper
func MarshalPayload(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

func (p *Peer) SendChan() <-chan signal.OutgoingMessage {
	return p.send
}
