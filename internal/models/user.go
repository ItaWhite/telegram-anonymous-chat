package models

type User struct {
	ID        int64
	State     int
	PartnerID int64
	Banned    bool
}

const (
	StateIdle = iota
	StateWaiting
	StatePaired
)
