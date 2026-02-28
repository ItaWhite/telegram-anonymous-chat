package models

import "time"

type User struct {
	ID         int64
	State      int
	PartnerID  int64
	Banned     bool
	Rating     int
	DailyChats int
	LastReset  time.Time
}

const (
	StateIdle = iota
	StateWaiting
	StatePaired
)
