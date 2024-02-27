package delivery

import (
	"net"
	"proxyServer/mongo/domain"
	"sync"
)

type Storage interface {
	Add(domain.HTTPTransaction) error
}

// A oneShotDialer implements net.Dialer whos Dial only returns a
// net.Conn as specified by c followed by an error for each subsequent Dial.
type oneShotDialer struct {
	targetConn net.Conn
	mu         sync.Mutex
}

// A oneShotListener implements net.Listener whos Accept only returns a
// net.Conn as specified by c followed by an error for each subsequent Accept.
type oneShotListener struct {
	clientConn net.Conn
}

type onCloseConn struct {
	net.Conn
	f func()
}
