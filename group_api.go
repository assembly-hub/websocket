// Package websocket
package websocket

type GroupAPI interface {
	Register(cli *Client)
	UnRegister(cli *Client)
	SendMsg(msg []byte)
}
