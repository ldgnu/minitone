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
//
// The list (items) and the play sequence (order) are kept separate: `order`
// maps a play position to an item ID.
//
//   - With shuffle off, order mirrors the list, so reordering items with
//     Move() reorders playback.
//   - With shuffle on, order is a fixed permutation of the item IDs. Adding or
//     removing a song appends/removes it *without* reshuffling what is already
//     queued — otherwise every keystroke-level change scrambled the sequence.
type Queue struct {
	mu      sync.Mutex
	items   []models.QueueItem
	order   []int64
	pos     int
	shuffle bool
	repeat  RepeatMode
	nextID  int64
}

func New() *Queue {
	return &Queue{pos: -1}
}

// Add appends a song to the end of the queue and returns its item ID.
func (q *Queue) Add(song models.Song) int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.addLocked(song)
}

func (q *Queue) addLocked(song models.Song) int64 {
	q.nextID++
	item := models.QueueItem{Song: song, Added: time.Now(), ID: q.nextID}
	q.items = append(q.items, item)
	q.order = append(q.order, item.ID) // new songs play last
	if q.pos < 0 {
		q.pos = 0
	}
	return item.ID
}

// AddUnique appends song unless an identical song is already queued.
// It reports whether the song was added.
func (q *Queue) AddUnique(song models.Song) bool {
	q.mu.Lock()
	key := song.Key()
	for _, it := range q.items {
		if it.Song.Key() == key {
			q.mu.Unlock()
			return false
		}
	}
	q.mu.Unlock()
	q.Add(song)
	return true
}

// AddNext inserts a song right after the current one (play-next).
func (q *Queue) AddNext(song models.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.nextID++
	item := models.QueueItem{Song: song, Added: time.Now(), ID: q.nextID}

	insertAt := len(q.items)
	if cur := q.currentItemLocked(); cur != nil {
		if i := q.itemIndexLocked(cur.ID); i >= 0 {
			insertAt = i + 1
		}
	}

	q.items = append(q.items, models.QueueItem{})
	copy(q.items[insertAt+1:], q.items[insertAt:])
	q.items[insertAt] = item

	// In the play sequence it goes right after the current track.
	insertPos := 0
	if q.pos >= 0 && q.pos+1 < len(q.order) {
		insertPos = q.pos + 1
	} else if len(q.order) > 0 {
		insertPos = len(q.order)
	}
	q.order = append(q.order, 0)
	copy(q.order[insertPos+1:], q.order[insertPos:])
	q.order[insertPos] = item.ID

	if q.pos < 0 {
		q.pos = 0
	}
}

func (q *Queue) Remove(index int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if index < 0 || index >= len(q.items) {
		return false
	}
	id := q.items[index].ID
	q.items = append(q.items[:index], q.items[index+1:]...)

	// Drop it from the play sequence, keeping pos on the same track.
	for i, oid := range q.order {
		if oid == id {
			switch {
			case i < q.pos:
				q.pos--
			case i == q.pos:
				// the removed track was playing: stay at the same slot, which
				// now holds the following track (or past the end).
			}
			q.order = append(q.order[:i], q.order[i+1:]...)
			break
		}
	}
	if len(q.items) == 0 {
		q.pos = -1
		q.order = nil
		return true
	}
	q.clampPosLocked()
	return true
}

// Move relocates the item at from so it ends up at index to in the list.
func (q *Queue) Move(from, to int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if from < 0 || from >= len(q.items) || to < 0 || to >= len(q.items) || from == to {
		return false
	}
	// Remember what is playing *before* touching the list: after the reorder
	// the same play position points at a different item.
	playingID := int64(-1)
	if cur := q.currentItemLocked(); cur != nil {
		playingID = cur.ID
	}

	item := q.items[from]
	rest := append(q.items[:from:from], q.items[from+1:]...) // shift-left copy
	rest = append(rest, models.QueueItem{})
	copy(rest[to+1:], rest[to:])
	rest[to] = item
	q.items = rest

	if !q.shuffle {
		// Without shuffle the play sequence follows the list, so rebuild it.
		q.rebuildLinearOrderLocked()
	}
	// Either way, playback stays on the same song.
	if playingID >= 0 {
		if pos := q.playPosLocked(playingID); pos >= 0 {
			q.pos = pos
		}
	}
	q.clampPosLocked()
	return true
}

// Current returns a copy of the current queue item, or nil.
func (q *Queue) Current() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	it := q.currentItemLocked()
	if it == nil {
		return nil
	}
	item := *it
	return &item
}

// Next advances according to shuffle/repeat and returns the new current item.
func (q *Queue) Next() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 || len(q.order) == 0 {
		return nil
	}

	switch q.repeat {
	case RepeatOne:
		if q.currentItemLocked() == nil {
			q.pos = 0
		}
	case RepeatAll:
		if q.pos < 0 {
			q.pos = 0
		} else {
			q.pos = (q.pos + 1) % len(q.order)
		}
	default:
		if q.pos < 0 {
			q.pos = 0
		} else if q.pos >= len(q.order)-1 {
			q.pos = -1
			return nil
		} else {
			q.pos++
		}
	}

	it := q.currentItemLocked()
	if it == nil {
		return nil
	}
	item := *it
	return &item
}

// PeekNext returns the next item without advancing (respecting repeat).
func (q *Queue) PeekNext() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 || len(q.order) == 0 {
		return nil
	}
	var idx int
	switch q.repeat {
	case RepeatOne:
		it := q.currentItemLocked()
		if it == nil {
			return nil
		}
		item := *it
		return &item
	case RepeatAll:
		idx = 0
		if q.pos >= 0 {
			idx = (q.pos + 1) % len(q.order)
		}
	default:
		if q.pos < 0 {
			idx = 0
		} else if q.pos >= len(q.order)-1 {
			return nil
		} else {
			idx = q.pos + 1
		}
	}
	it := q.byIDLocked(q.order[idx])
	if it == nil {
		return nil
	}
	item := *it
	return &item
}

// Prev steps back in the play order.
func (q *Queue) Prev() *models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 || len(q.order) == 0 {
		return nil
	}
	switch {
	case q.pos > 0:
		q.pos--
	case q.pos < 0:
		q.pos = 0
	case q.repeat == RepeatAll:
		q.pos = len(q.order) - 1
	}
	it := q.currentItemLocked()
	if it == nil {
		return nil
	}
	item := *it
	return &item
}

// SetCursor sets the current track by index into the items slice.
func (q *Queue) SetCursor(i int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if i < 0 || i >= len(q.items) {
		return false
	}
	if pos := q.playPosLocked(q.items[i].ID); pos >= 0 {
		q.pos = pos
		return true
	}
	return false
}

func (q *Queue) Items() []models.QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]models.QueueItem{}, q.items...)
}

// Songs returns the queued songs in list order.
func (q *Queue) Songs() []models.Song {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]models.Song, len(q.items))
	for i, it := range q.items {
		out[i] = it.Song
	}
	return out
}

// Cursor returns the index into the items slice of the current track, or -1.
func (q *Queue) Cursor() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	it := q.currentItemLocked()
	if it == nil {
		return -1
	}
	for i, candidate := range q.items {
		if candidate.ID == it.ID {
			return i
		}
	}
	return -1
}

// Len returns the number of queued items.
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

// SetShuffle toggles shuffle, keeping the current track playing.
// Turning shuffle ON reshuffles the upcoming sequence; turning it OFF restores
// the list order. Neither disturbs the track that is playing.
func (q *Queue) SetShuffle(v bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.shuffle == v || len(q.items) == 0 {
		q.shuffle = v
		return
	}
	cur := q.currentItemLocked()
	q.shuffle = v
	if v {
		ids := make([]int64, 0, len(q.items))
		for _, it := range q.items {
			ids = append(ids, it.ID)
		}
		rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		q.order = ids
	} else {
		q.rebuildLinearOrderLocked()
	}
	if cur != nil {
		if pos := q.playPosLocked(cur.ID); pos >= 0 {
			q.pos = pos
		}
	}
	q.clampPosLocked()
}

func (q *Queue) Repeat() RepeatMode {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.repeat
}

func (q *Queue) SetRepeat(r RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.repeat = r
}

// Clear removes every item from the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = nil
	q.order = nil
	q.pos = -1
}

// ── session support ────────────────────────────────────────────────────────

// Snapshot captures the queue for later restoration.
//
// It must not call the public accessors while holding the lock: those lock
// again and sync.Mutex is not reentrant (that deadlocks).
func (q *Queue) Snapshot() (songs []models.Song, cursor int, shuffle bool, repeat RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()

	songs = make([]models.Song, len(q.items))
	for i, it := range q.items {
		songs[i] = it.Song
	}
	// Resolve the cursor inline: calling the public Cursor() here would
	// deadlock on the non-reentrant mutex.
	cursor = -1
	if it := q.currentItemLocked(); it != nil {
		for i, candidate := range q.items {
			if candidate.ID == it.ID {
				cursor = i
				break
			}
		}
	}
	return songs, cursor, q.shuffle, q.repeat
}

// Restore replaces the queue contents with songs and restores preferences.
// cursor is an index into songs (ignored when out of range).
func (q *Queue) Restore(songs []models.Song, cursor int, shuffle bool, repeat RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = nil
	q.order = nil
	q.pos = -1
	q.nextID = 0
	for _, s := range songs {
		q.addLocked(s)
	}
	if shuffle {
		q.shuffle = true
		ids := make([]int64, 0, len(q.items))
		for _, it := range q.items {
			ids = append(ids, it.ID)
		}
		rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		q.order = ids
		q.pos = 0
	} else {
		q.rebuildLinearOrderLocked()
	}
	q.repeat = repeat
	if cursor >= 0 && cursor < len(q.items) {
		if pos := q.playPosLocked(q.items[cursor].ID); pos >= 0 {
			q.pos = pos
		}
	}
	q.clampPosLocked()
}

// ── internals ──────────────────────────────────────────────────────────────

func (q *Queue) currentItemLocked() *models.QueueItem {
	if q.pos < 0 || q.pos >= len(q.order) {
		return nil
	}
	return q.byIDLocked(q.order[q.pos])
}

func (q *Queue) byIDLocked(id int64) *models.QueueItem {
	for i := range q.items {
		if q.items[i].ID == id {
			return &q.items[i]
		}
	}
	return nil
}

// itemIndexLocked returns the list index of an item ID (-1 when absent).
func (q *Queue) itemIndexLocked(id int64) int {
	for i := range q.items {
		if q.items[i].ID == id {
			return i
		}
	}
	return -1
}

// playPosLocked maps an item ID to its play position (-1 when absent).
func (q *Queue) playPosLocked(id int64) int {
	for pos, oid := range q.order {
		if oid == id {
			return pos
		}
	}
	return -1
}

// rebuildLinearOrderLocked makes the play sequence follow the list order.
func (q *Queue) rebuildLinearOrderLocked() {
	q.order = make([]int64, len(q.items))
	for i, it := range q.items {
		q.order[i] = it.ID
	}
}

func (q *Queue) clampPosLocked() {
	if len(q.order) == 0 {
		q.pos = -1
		return
	}
	if q.pos < 0 || q.pos >= len(q.order) {
		q.pos = len(q.order) - 1
	}
}
