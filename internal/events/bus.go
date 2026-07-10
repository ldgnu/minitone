package events

import (
	"fmt"
	"sync"
)

type Type int

const (
	EventSongPlayed Type = iota
	EventSongPaused
	EventSongStopped
	EventSongResumed
	EventSongProgress
	EventVolumeChanged
	EventSearchStarted
	EventSearchCompleted
	EventSearchCancelled
	EventSearchCleared
	EventQueueChanged
	EventFavoriteToggled
	EventThemeChanged
	EventPlayerError
	EventLibraryUpdated
	EventShuffleToggled
	EventRepeatToggled
)

type Event struct {
	Type Type
	Data any
}

type Handler func(Event)

type Bus struct {
	subs map[Type][]Handler
	mu   sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{subs: make(map[Type][]Handler)}
}

func (b *Bus) Subscribe(t Type, fn Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[t] = append(b.subs[t], fn)
}

func (b *Bus) Unsubscribe(t Type, fn Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	handlers := b.subs[t]
	for i, h := range handlers {
		if Equal(h, fn) {
			b.subs[t] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

func (b *Bus) Emit(t Type, data any) {
	b.mu.RLock()
	handlers := append([]Handler{}, b.subs[t]...)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(Event{Type: t, Data: data})
	}
}

var globalBus *Bus
var globalOnce sync.Once

func Global() *Bus {
	globalOnce.Do(func() { globalBus = NewBus() })
	return globalBus
}

func Equal(a, b Handler) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}
