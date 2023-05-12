package multisub

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"

	inner "github.com/assembly-hub/websocket"
	"github.com/assembly-hub/websocket/config"
)

const (
	defaultRedisPubSubKeyPrefix = "ws_many_group_msg_prefix_"
)

type Manage struct {
	groupMap        map[string]*redisGroup
	pubSubKeyPrefix string
	r               *redis.Client
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
			group = newRedisGroup(m.r, groupName, m.pubSubKeyPrefix, m)
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
	group := m.groupMap[groupName]
	if group != nil {
		group.SendMsg([]byte(msg))
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

// AddGroupWithExt 每个组一个redis订阅
func (m *Manage) AddGroupWithExt(groupName string, w http.ResponseWriter, r *http.Request, ext *inner.GroupExtData) error {
	if groupName == "" {
		return fmt.Errorf("group name is empty")
	}

	conn, err := m.upgrade.Upgrade(w, r, nil)
	if err != nil {
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

func (m *Manage) SetUpgrade(up *websocket.Upgrader) {
	m.upgrade = up
}

func NewManager(r *redis.Client, label string) *Manage {
	if label == "" {
		label = defaultRedisPubSubKeyPrefix
	}

	m := &Manage{
		groupMap:        map[string]*redisGroup{},
		pubSubKeyPrefix: label,
		r:               r,
		mutex:           sync.Mutex{},
		groupMsgMaxLen:  1000,
		upgrade:         &config.WSDefaultUpdate,
	}
	return m
}
