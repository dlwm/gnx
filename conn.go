package gn

import (
	"github.com/alberliu/gn/buffer"
	"sync/atomic"
	"syscall"
	"time"
)

// Conn 客户端长连接
type Conn struct {
	s            *Server        // 服务器引用
	fd           int32          // 文件描述符
	addr         string         // 对端地址
	buffer       *buffer.Buffer // 读缓存区
	lastReadTime time.Time      // 最后一次读取数据的时间
	data         interface{}    // 业务自定义数据，用作扩展
}

// newConn 创建tcp链接
func newConn(fd int32, addr string, s *Server) *Conn {
	return &Conn{
		s:            s,
		fd:           fd,
		addr:         addr,
		buffer:       buffer.NewBuffer(s.readBufferPool.Get().([]byte)),
		lastReadTime: time.Now(),
	}
}

// GetFd 获取文件描述符
func (c *Conn) GetFd() int32 {
	return c.fd
}

// GetAddr 获取客户端地址
func (c *Conn) GetAddr() string {
	return c.addr
}

// Read 读取数据
func (c *Conn) Read() error {
	c.lastReadTime = time.Now()
	fd := int(c.GetFd())
	for {
		err := c.buffer.ReadFromFD(fd)
		if err != nil {
			// 缓存区暂无数据可读
			if err == syscall.EAGAIN {
				return nil
			}
			return err
		}

		err = c.s.decoder.Decode(c)
		if err != nil {
			return err
		}
	}
}

// Write 写入数据
func (c *Conn) Write(bytes []byte) (int, error) {
	return syscall.Write(int(c.fd), bytes)
}

// Close 关闭连接
func (c *Conn) Close() error {
	// 从epoll监听的文件描述符中删除
	err := c.s.epoll.RemoveAndClose(int(c.fd))
	if err != nil {
		log.Error(err)
	}

	// 从conns中删除conn
	c.s.conns.Delete(c.fd)
	// 归还缓存区
	c.s.readBufferPool.Put(c.buffer.Buf)
	// 连接数减一
	atomic.AddInt64(&c.s.connsNum, -1)
	return nil
}

// GetData 获取数据
func (c *Conn) GetData() interface{} {
	return c.data
}

// SetData 设置数据
func (c *Conn) SetData(data interface{}) {
	c.data = data
}
