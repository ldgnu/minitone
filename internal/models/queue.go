package models

import "time"

type QueueItem struct {
	Song  Song
	Added time.Time
	ID    int64
}
