//go:build windows
// +build windows

package gn

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"net"
	"syscall"
)

type epoll struct {
	listenFD int
	epollFD  int
	listener net.Listener
}

func newNetpoll(address string) (netpoll, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error listening:", err)
		return nil, err
	}

	fd, err := listener.(*net.TCPListener).File()
	if err != nil {
		fmt.Println("Error get fd:", err)
		return nil, err
	}

	iocpFD, err := windows.CreateIoCompletionPort(windows.Handle(fd.Fd()), 0, 0, 0)
	if err != nil {
		return nil, err
	}

	return &epoll{
		listenFD: int(fd.Fd()),
		epollFD:  int(iocpFD),
		listener: listener,
	}, err
}

func (n *epoll) accept() (nfd int, addr string, err error) {
	conn, err := n.listener.Accept()
	if err != nil {
		return
	}

	conn.TODO
	// 设置为非阻塞状态
	err = syscall.SetNonblock(nfd, true)
	if err != nil {
		return
	}

	err = syscall.EpollCtl(n.epollFD, syscall.EPOLL_CTL_ADD, nfd, &syscall.EpollEvent{
		Events: EpollRead,
		Fd:     int(nfd),
	})
	if err != nil {
		return
	}

	s := sa.(*syscall.SockaddrInet4)
	addr = fmt.Sprintf("%d.%d.%d.%d:%d", s.Addr[0], s.Addr[1], s.Addr[2], s.Addr[3], s.Port)
	return
}

func (n *epoll) addRead(fd int) error {
	err := syscall.EpollCtl(n.epollFD, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: EpollRead,
		Fd:     int(fd),
	})
	if err != nil {
		return err
	}
	return nil
}

func (n *epoll) getEvents() ([]event, error) {

	epollEvents := make([]syscall.EpollEvent, 100)
	num, err := syscall.EpollWait(n.epollFD, epollEvents, -1)
	if err != nil {
		return nil, err
	}

	events := make([]event, 0, len(epollEvents))
	for i := 0; i < num; i++ {
		event := event{
			FD: epollEvents[i].Fd,
		}
		if epollEvents[i].Events == EpollClose {
			event.Type = EventClose
		} else {
			event.Type = EventIn
		}
		events = append(events, event)
	}

	return events, nil
}

//var completionKey uintptr
//var bytesTransferred uint32
//var o *windows.Overlapped
//err := windows.GetQueuedCompletionStatus(
//windows.Handle(n.epollFD), &bytesTransferred, &completionKey, &o, syscall.INFINITE
//)
//if err != nil {
//fmt.Println("Error getting completion status:", err)
//return nil, err
//}

func (n *epoll) closeFD(fd int) error {
	// 移除文件描述符的监听 + 关闭文件描述符
	return windows.CloseHandle(windows.Handle(fd))
}

func (n *epoll) closeFDRead(fd int) error {
	return errors.New("TODO windows close fd read")
}

func (n *epoll) write(fd int, bytes []byte) (int, error) {
	return windows.Write(windows.Handle(fd), bytes)
}
