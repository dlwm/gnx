//go:build windows
// +build windows

package gn

import "fmt"

// TODO support iocp like epoll

func init() {
	fmt.Println("warning: suggest using Linux in production environment.")
}
