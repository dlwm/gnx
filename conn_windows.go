//go:build windows
// +build windows

package gnx

import (
	"github.com/dlwm/gnx/codec"
	"net"
	"sync/atomic"
)

// Conn 客户端长连接
type Conn struct {
	server *Server       // 服务器引用
	fd     int32         // 文件描述符
	addr   string        // 对端地址
	buffer *codec.Buffer // 读缓存区
	data   interface{}   // 业务自定义数据，用作扩展

	conn net.Conn
}

// newConn 创建tcp链接
func newConn(fd int32, addr string, server *Server, conn net.Conn) *Conn {
	return &Conn{
		server: server,
		fd:     fd,
		addr:   addr,
		buffer: codec.NewBuffer(server.readBufferPool.Get().([]byte)),

		conn: conn,
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

// GetBuffer 获取连接对应缓存区buffer
func (c *Conn) GetBuffer() *codec.Buffer {
	return c.buffer
}

// Read 读取数据
func (c *Conn) read() (err error) {
	if c.server.options.decoder == nil {
		c.server.handler.OnMessage(c, c.buffer.ReadAll())
	} else {
		var handle = func(bytes []byte) {
			c.server.handler.OnMessage(c, bytes)
		}
		err = c.server.options.decoder.Decode(c.buffer, handle)
		if err != nil {
			return err
		}
	}

	return err
}

// WriteWithEncoder 使用编码器写入
func (c *Conn) WriteWithEncoder(bytes []byte) error {
	return c.server.options.encoder.EncodeToWriter(c, bytes)
}

// Write 写入数据 todo 这里可能未能把所有数据写进去
func (c *Conn) Write(bytes []byte) (int, error) {
	return c.conn.Write(bytes)
}

// Close 关闭连接
func (c *Conn) Close() {
	// 从epoll监听的文件描述符中删除
	err := c.conn.Close()
	if err != nil {
		log.Error(err)
	}

	// 归还缓存区 (以是否存在缓存区为依据判断是否已经关闭)
	if c.buffer != nil {
		c.server.readBufferPool.Put(c.buffer.GetBuf())
		c.buffer = nil
		// 从conns中删除conn
		c.server.conns.Delete(c.addr)
		// 连接数减一
		atomic.AddInt64(&c.server.connsNum, -1)
	}
}

//// CloseRead 关闭连接
//func (c *Conn) CloseRead() error {
//	err := c.conn.Close()
//	if err != nil {
//		log.Error(err)
//	}
//	return nil
//}

// GetData 获取数据
func (c *Conn) GetData() interface{} {
	return c.data
}

// SetData 设置数据
func (c *Conn) SetData(data interface{}) {
	c.data = data
}
