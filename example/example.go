// Package websocket
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/assembly-hub/websocket"
	"github.com/assembly-hub/websocket/multisub"
	"github.com/assembly-hub/websocket/simplesub"
	"github.com/assembly-hub/websocket/singlesub"
)

var (
	rd *redis.Client
)

// SimpleGroupTest Demo
func SimpleGroupTest() {
	// fmt.Println("Hello, Goutil!")
	g := simplesub.NewManager()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := g.AddGroup("test", w, r)
		if err != nil {
			fmt.Println(err)
		}
	})

	go func() {
		for {
			time.Sleep(time.Second * 1)
			err := g.SendMsg("test", "123")
			if err != nil {
				panic(err)
			}
		}
	}()
}

// RedisGroupTest Demo
func RedisGroupTest() {
	// fmt.Println("Hello, Goutil!")
	g := multisub.NewManager(rd)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := g.AddGroup("test", w, r)
		if err != nil {
			fmt.Println(err)
		}
	})

	go func() {
		for {
			time.Sleep(time.Second * 1)
			err := g.SendMsg("test", "msg")
			if err != nil {
				panic(err)
			}
		}
	}()
}

// SingleRedisGroupTest Demo
func SingleRedisGroupTest() {
	// fmt.Println("Hello, Goutil!")
	g := singlesub.NewManager(rd)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := g.AddGroupWithExt("test", w, r, &websocket.GroupExtData{
			CloseSendData: "123",
			CloseCallback: func(data interface{}) {
				fmt.Println(data)
			},
		})
		if err != nil {
			fmt.Println(err)
		}
	})

	go func() {
		for {
			time.Sleep(time.Second * 1)
			err := g.SendMsg("test", "test")
			if err != nil {
				return
			}
		}
	}()
}

// SingleWSTest Demo
func SingleWSTest() {
	// fmt.Println("Hello, Goutil!")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		cli, err := websocket.NewWS(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}

		cli.SetDealMsg(func(msg []byte) []byte {
			fmt.Println(msg)
			return msg
		})

		go func(cli *websocket.Client) {
			for {
				time.Sleep(time.Second * 1)
				cli.SendMsg("test")
			}
		}(cli)
	})
}

func RunExample() {
	opts := redis.Options{}
	opts.Addr = "127.0.0.1:6379"
	opts.DB = 0

	rd = redis.NewClient(&opts)
	defer rd.Close()

	// ?????????
	// SimpleGroupTest()
	// ???????????????redis??????
	// RedisGroupTest()
	// ?????????????????????redis??????
	SingleRedisGroupTest()
	// ?????????ws??????
	// SingleWSTest()

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		panic(err)
	}
}
