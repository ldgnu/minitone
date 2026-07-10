package queue

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatAll
	RepeatOne
)

func (r RepeatMode) String() string {
	switch r {
	case RepeatAll:
		return "all"
	case RepeatOne:
		return "one"
	default:
		return "off"
	}
}

// Queue holds playable items with optional shuffle and repeat.
// pos is the index into order (play sequence). order maps play positions
// to item indices; when shuffle is off, order is the identity sequence.
type Queue struct {
	mu      sync.Mutex
	items   []models.QueueItem
	pos     int
	order   []int
	shuffle bool
	repeat  RepeatMode
	nextID  int64
}

func New() *Queue {
	return &Queue{pos: -1}
}

func (q *Queue) Add(song models.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.nextID++
	q.items = append(q.items, models.QueueItem{
		Song:  song,
		Added: time.Now(),
		ID:    q.nextID,
	})
	if q.pos < 0 {
		q.pos = 0
	}
	q.rebuildOrderLocked(false)
}

func (q *Queue) AddNext(song models.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.nextID++
	item := models.QueueItem{Song: song, Added: time.Now(), ID: q.nextID}

	// Insert after current play position in the items list.
	insertAt := 0
	if q.pos >= 0 && q.pos < len(q.order) {
		curIdx := q.order[q.pos]
		insertAt = curIdx + 1
	} else if len(q.items) > 0 {
		insertAt = len(q.items)
	}

	if insertAt >= len(q.items) {
		q.items = append(q.items, item)
	} else {
		q.items = append(q.items[:insertAt], append([]models.QueueItem{item}, q.items[insertAt:]...)...)
	}
	if q.pos < 0 {
		q.pos = 0
	}
	q.rebuildOrderLocked(false)
}

func (q *Queue) Remove(index int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if index < 0 || index >= len(q.items) {
		return false
	}
	q.items = append(q.items[:index], q.items[index+1:]...)
	if len(q.items) == 0 {
		q.pos = -1
		q.order = nil
		return true
	}
	q.rebuildOrderLocked(false)
	if q.pos >= len(q.order) {
		q.pos = len(q.order) - 1
	}
	return true
}

func (q *Queue) Move(from, to int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if from < 0 || from >= len(q.items) || to < 0 || to >= len(q.items) || from == to {
		return false
	}
	item := q.items[from]
	q.items = append(q.items[:from], q.items[from+1:]...)
	q.items = append(q.items[:to], append([]models.QueueItem{item}, q.items[to:]...)...)
	q.rebuildOrderLocked(false)
	return true
}

// Current returns a copy of the current queue item, or nil.
func (q *Queue) Current() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	idx := q.currentIndexLocked()
	if idx < 0 {
		return nil
	}
	item := q.items[idx]
	return &item
}

// Next advances according to shuffle/repeat and returns the new current item.
func (q *Queue) Next() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}

	switch q.repeat {
	case RepeatOne:
		idx := q.currentIndexLocked()
		if idx < 0 {
			q.pos = 0
			idx = q.order[0]
		}
		item := q.items[idx]
		return &item
	case RepeatAll:
		if q.pos < 0 {
			q.pos = 0
		} else {
			q.pos = (q.pos + 1) % len(q.order)
		}
	case RepeatOff:
		if q.pos < 0 {
			q.pos = 0
		} else if q.pos >= len(q.order)-1 {
			q.pos = -1
			return nil
		} else {
			q.pos++
		}
	}

	idx := q.currentIndexLocked()
	if idx < 0 {
		return nil
	}
	item := q.items[idx]
	return &item
}

// PeekNext returns the next item without advancing (respecting repeat).
func (q *Queue) PeekNext() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	switch q.repeat {
	case RepeatOne:
		idx := q.currentIndexLocked()
		if idx < 0 {
			return nil
		}
		item := q.items[idx]
		return &item
	case RepeatAll:
		nextPos := 0
		if q.pos >= 0 {
			nextPos = (q.pos + 1) % len(q.order)
		}
		item := q.items[q.order[nextPos]]
		return &item
	default:
		if q.pos < 0 {
			item := q.items[q.order[0]]
			return &item
		}
		if q.pos >= len(q.order)-1 {
			return nil
		}
		item := q.items[q.order[q.pos+1]]
		return &item
	}
}

func (q *Queue) Prev() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	if q.pos > 0 {
		q.pos--
	} else if q.pos < 0 {
		q.pos = 0
	} else if q.repeat == RepeatAll {
		q.pos = len(q.order) - 1
	}
	idx := q.currentIndexLocked()
	if idx < 0 {
		return nil
	}
	item := q.items[idx]
	return &item
}

// SetCursor sets the current item by index into the items slice.
func (q *Queue) SetCursor(i int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if i < 0 || i >= len(q.items) {
		return false
	}
	// Find play position that maps to item i.
	for pos, idx := range q.order {
		if idx == i {
			q.pos = pos
			return true
		}
	}
	q.pos = i
	return true
}

func (q *Queue) Items() []models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]models.QueueItem{}, q.items...)
}

// Cursor returns the index into the items slice of the current track, or -1.
func (q *Queue) Cursor() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.currentIndexLocked()
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue) Shuffle() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.shuffle
}

func (q *Queue) SetShuffle(v bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Remember current item so we can keep playing it.
	curIdx := q.currentIndexLocked()
	q.shuffle = v
	q.rebuildOrderLocked(true)
	if curIdx >= 0 {
		for pos, idx := range q.order {
			if idx == curIdx {
				q.pos = pos
				return
			}
		}
	}
}

func (q *Queue) Repeat() RepeatMode {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.repeat
}

func (q *Queue) SetRepeat(m RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.repeat = m
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = nil
	q.pos = -1
	q.order = nil
}

func (q *Queue) currentIndexLocked() int {
	if q.pos < 0 || q.pos >= len(q.order) || len(q.items) == 0 {
		return -1
	}
	idx := q.order[q.pos]
	if idx < 0 || idx >= len(q.items) {
		return -1
	}
	return idx
}

// rebuildOrderLocked rebuilds the play order.
// If reshuffle is true and shuffle is on, a new random permutation is used.
func (q *Queue) rebuildOrderLocked(reshuffle bool) {
	n := len(q.items)
	if n == 0 {
		q.order = nil
		return
	}
	if q.shuffle {
		if reshuffle || len(q.order) != n {
			q.order = rand.Perm(n)
		} else {
			// Keep existing permutation shape; rebuild identity then not needed.
			// After insert/remove, rebuild fresh permutation to stay consistent.
			q.order = rand.Perm(n)
		}
	} else {
		q.order = make([]int, n)
		for i := range q.order {
			q.order[i] = i
		}
	}
	if q.pos >= n {
		q.pos = n - 1
	}
}
