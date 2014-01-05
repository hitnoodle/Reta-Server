package db

import (
	"time"
)

type Activity struct {
	Player     string
	Version    string
	ServerTime time.Time
	Data       string
}

type Parameter struct {
	Key   string
	Value string
}

type Event struct {
	Player     string
	Version    string
	Action     string
	Date       time.Time
	Parameters []Parameter
}

type TimedEvent struct {
	Info     Event
	Duration time.Duration
}
