package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const defaultEventBuffer = 64

type EventHub struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan RuntimeEvent
	nextSubID   uint64
	nextEventID uint64
	buffer      int
}

func NewEventHub(buffer int) *EventHub {
	if buffer <= 0 {
		buffer = defaultEventBuffer
	}
	return &EventHub{
		subscribers: make(map[uint64]chan RuntimeEvent),
		buffer:      buffer,
	}
}

func (h *EventHub) Subscribe(ctx context.Context) <-chan RuntimeEvent {
	ch := make(chan RuntimeEvent, h.buffer)
	id := atomic.AddUint64(&h.nextSubID, 1)

	h.mu.Lock()
	h.subscribers[id] = ch
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.unsubscribe(id)
	}()

	return ch
}

func (h *EventHub) Publish(typ string, payload any) RuntimeEvent {
	event := RuntimeEvent{
		ID:        atomic.AddUint64(&h.nextEventID, 1),
		Type:      typ,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
	return event
}

func (h *EventHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, ch := range h.subscribers {
		delete(h.subscribers, id)
		close(ch)
	}
}

func (h *EventHub) unsubscribe(id uint64) {
	h.mu.Lock()
	ch, ok := h.subscribers[id]
	if ok {
		delete(h.subscribers, id)
		close(ch)
	}
	h.mu.Unlock()
}
