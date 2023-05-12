# websocket组件，包含：基础链接、三种组定义，满足99%的场景

[demo code](./example/example.go)

## 1、SimpleGroup
> 单节点的ws组，数据基于内存
```go
// 创建ws管理组
g := simplesub.NewManager()

// 添加ws进组
http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    err := g.AddGroup("test", w, r)
    if err != nil {
        fmt.Println(err)
    }
})

// 发送消息进组
go func() {
    for {
        time.Sleep(time.Second * 1)
        err := g.SendMsg("test", "123")
        if err != nil {
            panic(err)
        }
    }
}()
```

## 2、multisub group
> 基于redis，实现每个组一个独立的消息通道，适合大数据
```go
// label 每个建议不一样，防止数据干扰
g := multisub.NewManager(rd, "my_label")
// 设置自定义标签，防止冲突
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
```

## 3、SingleRedisGroup
> 基于redis，实现所有组共享消息通道，适合不太大的数据
```go
// label 每个建议不一样，防止数据干扰
g := singlesub.NewManager(rd, "my_label")
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
```

## 4、简单的ws链接
> 单一的链接
```go
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
```