// sfu/fanout.go
package sfu

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

// trackFanout — читает RTP пакеты от одного издателя
// и пишет их во все подключённые localTracks подписчиков.
// Потокобезопасен — подписчики добавляются в любой момент.
type trackFanout struct {
	publisherID string
	mu          sync.RWMutex
	tracks      []*webrtc.TrackLocalStaticRTP
}

func newTrackFanout(publisherID string) *trackFanout {
	return &trackFanout{publisherID: publisherID}
}

func (f *trackFanout) add(track *webrtc.TrackLocalStaticRTP) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tracks = append(f.tracks, track)
	log.Printf("[fanout %s] subscriber added, total: %d", f.publisherID, len(f.tracks))
}

func (f *trackFanout) remove(track *webrtc.TrackLocalStaticRTP) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.tracks[:0]
	for _, t := range f.tracks {
		if t != track {
			out = append(out, t)
		}
	}
	f.tracks = out
}

func (f *trackFanout) write(pkt []byte) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, t := range f.tracks {
		if _, err := t.Write(pkt); err != nil {
			// Подписчик ушёл — не падаем, просто логируем
			log.Printf("[fanout %s] write error: %v", f.publisherID, err)
		}
	}
}
