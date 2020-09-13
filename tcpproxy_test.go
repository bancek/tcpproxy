package tcpproxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestProxy(t *testing.T) {
	var handler http.HandlerFunc

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
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

	doneCh := make(chan struct{}, 1)

	handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// we need to write a few KB for http server to flush
		_, _ = w.Write(make([]byte, 100*1024))

		time.Sleep(500 * time.Millisecond)

		if err := proxy.Close(); err != nil {
			t.Fatal(err)
		}

		time.Sleep(500 * time.Millisecond)

		_, _ = w.Write([]byte("test"))

		doneCh <- struct{}{}
	}

	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err == nil {
		t.Fatal("expected err not to be nil")
	}

	<-doneCh

	err = proxy.Start()
	if err != nil {
		t.Fatal(err)
	}

	handler = func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	}

	resp, err = http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte("test")) {
		t.Fatal("expected b to equal \"test\"")
	}
}
