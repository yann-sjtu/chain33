package types

import (
	"errors"
	"time"
)

var (
	ErrLength             = errors.New("length not equal")
	ErrProtocolNotSupport = errors.New("protocol not support")
	ErrNotFound           = errors.New("not found")
	ErrInvalidParam       = errors.New("invalid parameter")
	ErrInvalidMessage     = errors.New("invalid message")
	ErrInvalidResponse    = errors.New("invalid response")

	ExpiredTime     = time.Hour * 24
	RefreshInterval = time.Hour * 4
)
