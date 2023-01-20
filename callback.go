// Package websocket
package websocket

type GroupExtData struct {
	CloseSendData interface{}
	CloseCallback func(data interface{})
	ReceiveMsg    func(msg []byte) []byte
}
