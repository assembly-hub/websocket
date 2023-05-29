// Package websocket
package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/assembly-hub/websocket/config"
	"github.com/assembly-hub/websocket/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
)

// Client is a middleman between the websocket connection and the group.
type Client struct {
	GroupName string
	Group     GroupAPI

	initData      interface{}
	closeCallback func(data interface{})

	// The websocket connection.
	Conn *websocket.Conn
	// Buffered channel of outbound messages.
	Send chan []byte

	// 定义数据处理函数
	dealWithMsg func(msg []byte) []byte
}

func (c *Client) SetCloseCallback(f func(data interface{})) {
	c.closeCallback = f
}

func (c *Client) SetData(data interface{}) {
	c.initData = data
}

func (c *Client) Close() {
	defer func() {
		if p := recover(); p != nil {
			log.Log.Error(context.Background(), fmt.Sprintf("%v", p))
		}
	}()

	err := c.Conn.Close()
	if err != nil {
		log.Log.Error(context.Background(), err.Error())
	}
}

func (c *Client) readData() {
	defer func() {
		if c.Group != nil {
			c.Group.UnRegister(c)
		}
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Log.Error(context.Background(), err.Error())
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Log.Error(context.Background(), err.Error())
			}
			if _, ok := err.(*websocket.CloseError); ok {
				if c.closeCallback != nil {
					go c.closeCallback(c.initData)
				}
			}
			break
		}
		// message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		if c.dealWithMsg != nil {
			message = c.dealWithMsg(message)
			if message == nil {
				continue
			}
		}
		if c.Group != nil {
			err := c.Group.SendMsg(message)
			if err != nil {
				log.Log.Error(context.Background(), err.Error())
			}
		} else {
			c.Send <- message
		}
	}
}

func (c *Client) SetDealMsg(f func(msg []byte) []byte) {
	c.dealWithMsg = f
}

func (c *Client) writeData() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		err := c.Conn.Close()
		if err != nil {

		}
	}()
	for {
		select {
		case message, ok := <-c.Send:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Log.Error(context.Background(), err.Error())
			}
			if !ok {
				// The group closed the channel.
				err = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Log.Error(context.Background(), err.Error())
				}
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Log.Error(context.Background(), err.Error())
				return
			}
			_, err = w.Write(message)
			if err != nil {
				log.Log.Error(context.Background(), err.Error())
				return
			}

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, err = w.Write(newline)
				if err != nil {
					log.Log.Error(context.Background(), err.Error())
				}
				_, err = w.Write(<-c.Send)
				if err != nil {
					log.Log.Error(context.Background(), err.Error())
				}
			}

			if err = w.Close(); err != nil {
				log.Log.Error(context.Background(), err.Error())
				return
			}
		case <-ticker.C:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Log.Error(context.Background(), err.Error())
			}
			if err = c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Log.Error(context.Background(), err.Error())
				return
			}
		}
	}
}

func (c *Client) SendMsg(msg string) {
	c.Send <- []byte(msg)
}

func (c *Client) Run() {
	go c.readData()
	go c.writeData()
}

func NewWS(w http.ResponseWriter, r *http.Request, upgrade *websocket.Upgrader) (*Client, error) {
	if upgrade == nil {
		upgrade = &config.WSDefaultUpdate
	}
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		GroupName: "",
		Group:     nil,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	client.Run()
	return client, nil
}
