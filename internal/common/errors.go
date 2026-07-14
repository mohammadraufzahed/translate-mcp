package common

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrProviderUnavailable = Error("provider unavailable")
	ErrRateLimited         = Error("rate limited")
	ErrInvalidInput        = Error("invalid input")
	ErrNotFound            = Error("not found")
)
