package event

import (
	"context"
	"sync"
)

type Handler func(ctx context.Context, ev Event) error

type Bus interface {
	Publish(ctx context.Context, ev Event) error
	Subscribe(t Type, handler Handler) (unsubscribe func())
}

type MemoryBus struct {
	mu        sync.RWMutex
	handlers  map[Type][]Handler
	bufSize   int
	dropCount int64
}

func NewMemoryBus(subscriberBuffer int) *MemoryBus {
	if subscriberBuffer <= 0 {
		subscriberBuffer = 64
	}
	return &MemoryBus{
		handlers: map[Type][]Handler{},
		bufSize:  subscriberBuffer,
	}
}

func (b *MemoryBus) Publish(ctx context.Context, ev Event) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[ev.EventType()]...)
	b.mu.RUnlock()

	for _, h := range handlers {
		handler := h
		go func() {
			_ = handler(ctx, ev)
		}()
	}
	return nil
}

func (b *MemoryBus) Subscribe(t Type, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[t] = append(b.handlers[t], handler)
	idx := len(b.handlers[t]) - 1
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.handlers[t]
		if idx < 0 || idx >= len(list) {
			return
		}
		b.handlers[t] = append(list[:idx], list[idx+1:]...)
	}
}

func (b *MemoryBus) DropCount() int64 {
	return b.dropCount
}
