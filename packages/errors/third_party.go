package errors

import "errors"

var (
	ErrThirdPartyRateLimited = errors.New("third party api returned 429 too many requests")
	ErrThirdPartyServer    = errors.New("third party api returned 5xx server error")
)