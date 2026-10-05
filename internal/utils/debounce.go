package utils

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu      sync.Mutex
	timer   *time.Timer
	pending bool
	delay   time.Duration
}

func NewDebouncer(delay time.Duration) *Debouncer {
	if delay <= 0 {
		delay = 300 * time.Millisecond
	}
	return &Debouncer{delay: delay}
}

// SetDelay changes the debounce window for subsequent Reset calls.
func (d *Debouncer) SetDelay(delay time.Duration) {
	if delay <= 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.delay = delay
}

// Delay returns the current debounce window.
func (d *Debouncer) Delay() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.delay
}

func (d *Debouncer) Reset(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.pending = true
	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		d.pending = false
		d.mu.Unlock()
		fn()
	})
}

func (d *Debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.pending = false
}

func (d *Debouncer) Pending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pending
}
