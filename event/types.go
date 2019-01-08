package event

import "errors"

// errors
var (
	ErrWrongEventTopic = errors.New("required event topic doesn't exist in topic list")
)
