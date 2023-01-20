package singlesub

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"

	inner "github.com/assembly-hub/websocket"
	"github.com/assembly-hub/websocket/config"
)

const (
	defaultPubSubKeyPrefix = "ws_group_msg_prefix_"
)

type Manage struct {
	groupMap        map[string]*redisGroup
	redis           *redis.Client
	pubSub          *redis.PubSub
	pubSubKeyPrefix string
	mutex           sync.Mutex
	groupMsgMaxLen  int
	upgrade         *websocket.Upgrader
}

func (m *Manage) addGroup(groupName string, conn *websocket.Conn, ext *inner.GroupExtData) {
	var group *redisGroup
	if gp, ok := m.groupMap[groupName]; !ok {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if gp, ok = m.groupMap[groupName]; !ok {
			group = newRedisGroup(groupName, m)
			newMap := map[string]*redisGroup{
				groupName: group,
			}
			for k, v := range m.groupMap {
				newMap[k] = v
			}
			m.groupMap = newMap
		} else {
			group = gp
		}
	} else {
		group = gp
	}

	c := &inner.Client{
		GroupName: groupName,
		Group:     group,
		Conn:      conn,
		Send:      make(chan []byte, m.groupMsgMaxLen*3),
	}

	if ext != nil {
		c.SetDealMsg(ext.ReceiveMsg)
		c.SetData(ext.CloseSendData)
		c.SetCloseCallback(ext.CloseCallback)
	}

	group.Register(c)

	c.Run()
}

func (m *Manage) sendMsg(groupName string, msg string) {
	if m.redis == nil {
		return
	}

	r, ctx := m.redis, m.redis.Context()
	err := r.Publish(ctx, m.pubSubKeyPrefix+groupName, msg).Err()
	if err != nil {
		log.Println(err)
	}
}

func (m *Manage) MsgSub() {
	r, ctx := m.redis, m.redis.Context()
	m.pubSub = r.PSubscribe(ctx, m.pubSubKeyPrefix+"*")
	_, err := m.pubSub.Receive(ctx)
	if err != nil {
		log.Println(err)
		return
	}

	ch := m.pubSub.Channel()
	for msg := range ch {
		groupName := msg.Channel[len(m.pubSubKeyPrefix):]
		msgStr := msg.Payload
		group := m.groupMap[groupName]
		if group != nil {
			group.sendData([]byte(msgStr))
		}
	}
}

func (m *Manage) delGroup(groupName string) {
	if _, ok := m.groupMap[groupName]; ok {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if _, ok = m.groupMap[groupName]; ok {
			newMap := map[string]*redisGroup{}
			for k, v := range m.groupMap {
				if k == groupName {
					continue
				}
				newMap[k] = v
			}
			m.groupMap = newMap
		}
	}
}

// AddGroupWithExt 只一个redis订阅
func (m *Manage) AddGroupWithExt(groupName string, w http.ResponseWriter, r *http.Request, ext *inner.GroupExtData) error {
	if groupName == "" {
		return fmt.Errorf("group name is empty")
	}

	conn, err := m.upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	m.addGroup(groupName, conn, ext)
	return nil
}

func (m *Manage) AddGroup(groupName string, w http.ResponseWriter, r *http.Request) error {
	return m.AddGroupWithExt(groupName, w, r, nil)
}

func (m *Manage) SendMsg(groupName string, msg string) error {
	if groupName == "" {
		return fmt.Errorf("group name is empty")
	}

	m.sendMsg(groupName, msg)
	return nil
}

func (m *Manage) SetMaxMsgLength(n int) {
	m.groupMsgMaxLen = n
}

func (m *Manage) SetLabel(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("label cannot be empty or blank")
	}

	m.pubSubKeyPrefix = s
	return nil
}

func (m *Manage) SetUpgrade(up *websocket.Upgrader) {
	m.upgrade = up
}

func NewManager(r *redis.Client) *Manage {
	m := &Manage{
		redis:           r,
		groupMap:        map[string]*redisGroup{},
		pubSubKeyPrefix: defaultPubSubKeyPrefix,
		mutex:           sync.Mutex{},
		groupMsgMaxLen:  1000,
		upgrade:         &config.WSDefaultUpdate,
	}
	go m.MsgSub()
	return m
}
