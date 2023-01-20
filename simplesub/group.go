// Package simplesub
package simplesub

import (
	"github.com/assembly-hub/websocket"
)

// simpleGroup maintains the set of active clients and broadcasts messages to the
// clients.
type simpleGroup struct {
	// Registered clients.
	clients map[*websocket.Client]struct{}

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *websocket.Client

	// Unregister requests from clients.
	unregister chan *websocket.Client

	// ws manager
	m *Manage
}

func (g *simpleGroup) SendMsg(msg []byte) {
	g.broadcast <- msg
}

func (g *simpleGroup) Run() {
	for {
		select {
		case c := <-g.register:
			g.clients[c] = struct{}{}
		case c := <-g.unregister:
			if _, ok := g.clients[c]; ok {
				delete(g.clients, c)
				close(c.Send)
			}
			if len(g.clients) <= 0 {
				g.m.delGroup(c.GroupName)
			}
		case message := <-g.broadcast:
			for c := range g.clients {
				select {
				case c.Send <- message:
				default:
					g.UnRegister(c)
					c.Close()
				}
			}
		}
	}
}

func (g *simpleGroup) Register(cli *websocket.Client) {
	g.register <- cli
}

func (g *simpleGroup) UnRegister(cli *websocket.Client) {
	g.unregister <- cli
}

func newSimpleGroup(m *Manage) *simpleGroup {
	g := &simpleGroup{
		// Registered clients.
		clients: map[*websocket.Client]struct{}{},

		// Inbound messages from the clients.
		broadcast: make(chan []byte, m.groupMsgMaxLen),

		// Register requests from the clients.
		register: make(chan *websocket.Client),

		// Unregister requests from clients.
		unregister: make(chan *websocket.Client),

		m: m,
	}
	go g.Run()
	return g
}
