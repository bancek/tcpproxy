package tcpproxy

import (
	"net"
	"sync"
	"time"

	"inet.af/tcpproxy"
)

// Proxy is a TCP proxy. Wrapper around https://inet.af/tcpproxy
type Proxy struct {
	Proxy     *tcpproxy.Proxy
	DialProxy *tcpproxy.DialProxy

	listenAddr       string
	connections      map[net.Conn]struct{}
	connectionsMutex sync.Mutex
	onServerRead     func(c net.Conn, b []byte) (n int, err error)
	onServerWrite    func(c net.Conn, b []byte) (n int, err error)
}

// New creates a new proxy instance.
func New(listenAddr string, toAddr string) *Proxy {
	dialProxy := &tcpproxy.DialProxy{Addr: toAddr}

	p := &Proxy{
		Proxy:     nil,
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
	conn = &connWrapper{
		proxy: p,
		conn:  conn,
	}

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
	err := p.Proxy.Close()

	p.Proxy = nil

	p.CloseConnections()

	return err
}

// SetOnServerRead sets TCP connection Read interceptor. Called when client calls Write.
func (p *Proxy) SetOnServerRead(onServerRead func(c net.Conn, b []byte) (n int, err error)) {
	p.onServerRead = onServerRead
}

// SetOnServerWrite sets TCP connection Write interceptor. Called when client calls Read.
func (p *Proxy) SetOnServerWrite(onServerWrite func(c net.Conn, b []byte) (n int, err error)) {
	p.onServerWrite = onServerWrite
}

type connWrapper struct {
	proxy *Proxy
	conn  net.Conn
}

func (c *connWrapper) Read(b []byte) (n int, err error) {
	if c.proxy.onServerRead != nil {
		return c.proxy.onServerRead(c.conn, b)
	}
	return c.conn.Read(b)
}

func (c *connWrapper) Write(b []byte) (n int, err error) {
	if c.proxy.onServerWrite != nil {
		return c.proxy.onServerWrite(c.conn, b)
	}
	return c.conn.Write(b)
}

func (c *connWrapper) Close() error {
	return c.conn.Close()
}

func (c *connWrapper) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *connWrapper) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connWrapper) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *connWrapper) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *connWrapper) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
