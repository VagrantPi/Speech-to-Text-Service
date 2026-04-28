package llm

import "errors"

var ErrRateLimited = errors.New("rate limited by global redis token bucket")

type RateLimitedError struct{}

func (e *RateLimitedError) Error() string {
	return "rate limited"
}

func (e *RateLimitedError) IsRateLimited() bool {
	return true
}