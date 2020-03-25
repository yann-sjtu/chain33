package types

import (
	"errors"
	"time"
)

//Message 协议传递时统一消息类型
// Protocol指定具体协议
// Data是消息体
type Message struct {
	ProtocolID string
	Params     interface{}
}

type Response struct {
	Result interface{}
	Error  error
}

type StorageData struct {
	Data        interface{}
	RefreshTime time.Time
}

var (
	ErrLength             = errors.New("length not equal")
	ErrProtocolNotSupport = errors.New("protocol not support")
	ErrNotFound           = errors.New("not found")
	ErrInvalidParam       = errors.New("invalid parameter")
	ErrInvalidResponse    = errors.New("invalid response")
	ErrUnexpected         = errors.New("unexpected error")
	ErrDataExpired        = errors.New("data expired")
	ErrDBSave             = errors.New("save data to DB error")

	ExpiredTime     = time.Hour * 24
	RefreshInterval = time.Hour * 4
)
