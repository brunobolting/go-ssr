package user

import "errors"

var ErrUserNotFound = errors.New("user not found")

var ErrRoleNotFound = errors.New("role not found")

var ErrUserAlreadyExists = errors.New("user already exists")

var ErrInvalidRequest = errors.New("invalid request")

var ErrInvalidEmailOrPassword = errors.New("email or password invalid")
