package auth

import "errors"

var (
	ErrNoValidatedClaims = errors.New("no validated claims found in request ctx")
	ErrInvalidSubject    = errors.New("invalid subject claim")
)
