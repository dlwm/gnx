//go:build !windows
// +build !windows

package gnx

type netpoll interface {
	accept() (nfd int, addr string, err error)
	closeFD(fd int) error
	getEvents() ([]event, error)
	closeFDRead(fd int) error

	write(fd int, bytes []byte) (int, error)
}
