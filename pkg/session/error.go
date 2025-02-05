package session

import "errors"

var ErrSessionNotFound = errors.New("session not found")

var ErrSessionExpired = errors.New("session expired")

var ErrUserUnauthorized = errors.New("unauthorized")

var ErrInvalidSession = errors.New("invalid session")

var ErrUserForbidden = errors.New("you don't have permission to access this resource")
