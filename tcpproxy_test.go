package tcpproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	proxy, err := NewUnusedAddr("127.0.0.1", u.Host)
	if err != nil {
		t.Fatal(err)
	}
	err = proxy.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()

	u.Host = proxy.ListenAddr()

	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	proxy.Close()

	_, err = resp.Body.Read([]byte{0})
	if err == nil {
		t.Fatal("expected err not to be nil")
	}
}
