// Package config
package config

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type WSUpdateConf = websocket.Upgrader

// WSDefaultUpdate 默认升级配置
var WSDefaultUpdate = WSUpdateConf{
	HandshakeTimeout: time.Second * 10,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func SetCheckOrigin(f func(r *http.Request) bool) {
	WSDefaultUpdate.CheckOrigin = f
}

func Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	return WSDefaultUpdate.Upgrade(w, r, responseHeader)
}
