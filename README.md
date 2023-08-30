# gn
### 简述
gn是一个基于linux下epoll的网络框架，目前只对Linux环境下实现epoll处理，windows仅做支持，gn可以配置处理网络事件的goroutine数量，相比golang原生库，在海量链接下，可以减少goroutine的开销，从而减少系统资源占用。
### 支持功能
1.tcp拆包粘包  
支持多种编解码方式，使用sync.pool申请读写使用的字节数组，减少内存申请开销以及GC压力。  
2.客户端超时踢出  
可以设置超时时间，gn会定时检测超出超时的TCP连接（在指定时间内没有发送数据的连接）,进行释放。
### 使用方式
```go
package main

import (
	gn "github.com/dlwm/gnx"
	"github.com/dlwm/gnx/codec"
	"net"
	"strconv"
	"time"
)

var (
	decoder = codec.NewUvarintDecoder()
	encoder = codec.NewUvarintEncoder(1024)
)

var log = gn.GetLogger()

type Handler struct{}

func (*Handler) OnConnect(c *gn.Conn) {
	log.Info("server:connect:", c.GetFd(), c.GetAddr())
}
func (*Handler) OnMessage(c *gn.Conn, bytes []byte) {
	c.WriteWithEncoder(bytes)
	log.Info("server:read:", string(bytes))
}
func (*Handler) OnClose(c *gn.Conn, err error) {
	log.Info("server:close:", c.GetFd(), err)
}

func startServer() {
	server, err := gn.NewServer(":8080", &Handler{},
		gn.WithDecoder(decoder),
		gn.WithEncoder(encoder),
		gn.WithTimeout(5*time.Second),
		gn.WithReadBufferLen(10))
	if err != nil {
		log.Info("err")
		return
	}

	server.Run()
}
```
