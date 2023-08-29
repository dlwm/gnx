//go:build windows
// +build windows

package gn

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrReadTimeout = errors.New("tcp read timeout")
	ErrServerDown  = errors.New("tcp server down")
)

// Handler Server 注册接口
type Handler interface {
	OnConnect(c *Conn)               // OnConnect 当TCP长连接建立成功是回调
	OnMessage(c *Conn, bytes []byte) // OnMessage 当客户端有数据写入是回调
	OnClose(c *Conn, err error)      // OnClose 当客户端主动断开链接或者超时时回调,err返回关闭的原因
}

const (
	EventIn      = 1 // 数据流入
	EventClose   = 2 // 断开连接
	EventTimeout = 3 // 检测到超时
)

// Server TCP服务
type Server struct {
	listener       net.Listener
	options        *options   // 服务参数
	readBufferPool *sync.Pool // 读缓存区内存池
	handler        Handler    // 注册的处理
	conns          sync.Map   // TCP长连接管理
	connsNum       int64      // 当前建立的长连接数量
	stop           chan int   // 服务器关闭信号
}

// NewServer 创建server服务器
func NewServer(address string, handler Handler, opts ...Option) (*Server, error) {
	options := getOptions(opts...)

	// 初始化读缓存区内存池
	readBufferPool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, options.readBufferLen)
			return b
		},
	}

	// 启用监听
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &Server{
		listener:       listener,
		options:        options,
		readBufferPool: readBufferPool,
		handler:        handler,
		conns:          sync.Map{},
		connsNum:       0,
		stop:           make(chan int),
	}, nil
}

// GetConn 获取Conn
func (s *Server) GetConn(addr string) (*Conn, bool) {
	value, ok := s.conns.Load(addr)
	if !ok {
		return nil, false
	}
	return value.(*Conn), true
}

//func (s *Server) GetConn(fd int) (*Conn, bool) {
//	value, ok := s.conns.Load(fd)
//	if !ok {
//		return nil, false
//	}
//	return value.(*Conn), true
//}

// GetConnsNum 获取当前长连接的数量
func (s *Server) GetConnsNum() int64 {
	return atomic.LoadInt64(&s.connsNum)
}

// Run 启动服务
func (s *Server) Run() {
	log.Info("gn server run")
	s.startAccept()
}

// Stop 启动服务
func (s *Server) Stop() {
	_ = s.listener.Close()
	close(s.stop)
}

// startAccept 开始接收连接请求
func (s *Server) startAccept() {
	for i := 0; i < s.options.acceptGNum; i++ {
		go s.accept()
	}
	log.Info(fmt.Sprintf("start accept by %d goroutine", s.options.acceptGNum))
}

// accept 接收连接请求
func (s *Server) accept() {
	for {
		select {
		case <-s.stop:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Error(err)
				continue
			}

			//file, err := conn.(*net.TCPConn).File()
			//if err != nil {
			//	log.Error(err)
			//	continue
			//}
			//fd := int(file.Fd())
			addr := conn.RemoteAddr().String()
			gnConn := newConn(0, addr, s, conn)
			s.conns.Store(addr, gnConn)
			atomic.AddInt64(&s.connsNum, 1)
			s.handler.OnConnect(gnConn)

			// 读取协程
			go handleConn(gnConn)
		}
	}
}

func handleConn(c *Conn) {
	defer c.Close() // 顺带通知协程 conn 已关闭
	timer := time.NewTimer(c.server.options.timeout)
	gotRead := make(chan struct{})

	// 读取协程
	go func() {
		defer c.Close()
		for {
			if c.GetBuffer() == nil {
				return
			}
			_, err := c.buffer.ReadFromReader(c.conn)
			if err != nil {
				if err == io.EOF {
					c.server.handler.OnClose(c, err)
					break
				}
				log.Error(err)
				c.server.handler.OnClose(c, err)
				break
			}
			if err = c.read(); err != nil {
				log.Error(err)
				break
			}
			gotRead <- struct{}{}
		}
	}()

	// 超时监测
	for {
		select {
		case <-c.server.stop:
			c.server.handler.OnClose(c, ErrServerDown)
			break
		case <-timer.C:
			c.server.handler.OnClose(c, ErrReadTimeout)
			break
		case <-gotRead:
			timer.Reset(c.server.options.timeout)
		}
	}
}
