package tcpproxy

import (
	"net"
	"sync"

	"inet.af/tcpproxy"
)

// Proxy is a TCP proxy. Wrapper arround inet.af/tcpproxy
type Proxy struct {
	Proxy     *tcpproxy.Proxy
	DialProxy *tcpproxy.DialProxy

	listenAddr       string
	connections      map[net.Conn]struct{}
	connectionsMutex sync.Mutex

	proxyMutex sync.Mutex
}

// New creates a new proxy instance.
func New(listenAddr string, toAddr string) *Proxy {
	dialProxy := &tcpproxy.DialProxy{Addr: toAddr}

	p := &Proxy{
		DialProxy: dialProxy,

		listenAddr:  listenAddr,
		connections: map[net.Conn]struct{}{},
	}

	return p
}

// NewUnusedAddr creates a new proxy instance on a unused random port.
func NewUnusedAddr(listenHost string, toAddr string) (*Proxy, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(listenHost, "0"))
	if err != nil {
		return nil, err
	}
	listener.Close()

	listenAddr := listener.Addr().(*net.TCPAddr).String()

	return New(listenAddr, toAddr), nil
}

// ListenAddr returns the listen address.
func (p *Proxy) ListenAddr() string {
	return p.listenAddr
}

// HandleConn is used for hooking into inet.af/tcpproxy
func (p *Proxy) HandleConn(conn net.Conn) {
	p.connectionsMutex.Lock()
	p.connections[conn] = struct{}{}
	p.connectionsMutex.Unlock()

	defer func() {
		p.connectionsMutex.Lock()
		delete(p.connections, conn)
		p.connectionsMutex.Unlock()
	}()

	p.DialProxy.HandleConn(conn)
}

// Start starts the proxy.
func (p *Proxy) Start() error {
	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	if p.Proxy != nil {
		return nil
	}

	p.Proxy = &tcpproxy.Proxy{}

	p.Proxy.AddRoute(p.listenAddr, p)

	return p.Proxy.Start()
}

// CloseConnections closes the currently active connections.
func (p *Proxy) CloseConnections() {
	p.connectionsMutex.Lock()
	connections := make([]net.Conn, 0, len(p.connections))
	for conn := range p.connections {
		connections = append(connections, conn)
	}
	p.connectionsMutex.Unlock()

	for _, conn := range connections {
		conn.Close()
	}
}

// Close closes the TCP listener and closes all active connections.
func (p *Proxy) Close() error {
	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	if p.Proxy == nil {
		return nil
	}

	err := p.Proxy.Close()

	p.CloseConnections()

	p.Proxy = nil

	return err
}
