package protocol

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/queue"
)

//type Creator func(host Global) Protocol
//
//var (
//	protocolMap   = make(map[protocol.ID]Creator)
//	protocolMutex sync.RWMutex
//)
//
//func Register(id protocol.ID, f func(global Global) Protocol) {
//	protocolMutex.Lock()
//	defer protocolMutex.Unlock()
//	if _, ok := protocolMap[id]; ok {
//		panic("dup register manager, id:" + string(id))
//	}
//	protocolMap[id] = f
//}
//
//func Load(id protocol.ID) Creator {
//	protocolMutex.RLock()
//	defer protocolMutex.RUnlock()
//	return protocolMap[id]
//}
//
//func FetchAll() map[protocol.ID]Creator {
//	protocolMutex.RLock()
//	defer protocolMutex.RUnlock()
//	//返回一个protocolMap的副本，防止外部修改
//	m := make(map[protocol.ID]Creator)
//	for id, p := range protocolMap {
//		m[id] = p
//	}
//	return m
//}

// EventHandler handle chain33 event
type EventHandler func(*queue.Message)

var (
	eventHandlerMap   = make(map[int64]EventHandler)
	eventHandlerMutex sync.RWMutex
)

// RegisterEventHandler 注册消息处理函数
func RegisterEventHandler(eventID int64, handler EventHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	eventHandlerMutex.Lock()
	defer eventHandlerMutex.Unlock()
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d", eventID))
	}
	eventHandlerMap[eventID] = handler
}

// GetEventHandler get event handler
func GetEventHandler(eventID int64) (EventHandler, bool) {
	eventHandlerMutex.RLock()
	defer eventHandlerMutex.RUnlock()
	handler, ok := eventHandlerMap[eventID]

	return handler, ok
}

// ClearEventHandler clear event handler map
func ClearEventHandler() {
	eventHandlerMutex.RLock()
	defer eventHandlerMutex.RUnlock()
	eventHandlerMap = make(map[int64]EventHandler)
}
