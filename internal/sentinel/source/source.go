package source

import "time"

type Event struct {
	IP        string
	Site      string
	Status    int
	Method    string
	Path      string
	UserAgent string
	When      time.Time
}

type Backend interface {
	Read(offset int64) ([]Event, int64, error)
}
