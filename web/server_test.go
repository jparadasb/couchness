package web

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	originalDB := storage.Db
	originalConfig := storage.AppConfiguration
	t.Cleanup(func() {
		storage.Db = originalDB
		storage.AppConfiguration = originalConfig
	})

	db, err := storage.New(filepath.Join(dir, "database"), nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.Db = db
	storage.AppConfiguration = &models.AppConfiguration{}
}

func TestNewRejectsAuthWithoutColon(t *testing.T) {
	_, err := New(Options{Auth: "nocolon"})
	if err == nil {
		t.Fatal("expected error for auth without colon")
	}
}

func TestHandlerAuthRejectsAndAccepts(t *testing.T) {
	setupTestDB(t)
	s, err := New(Options{Auth: "admin:pw", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptestOrFatal(t, http.MethodGet, "/shows", nil)
	rr := serve(s.Handler(), req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != `Basic realm="couchness"` {
		t.Fatalf("expected WWW-Authenticate, got %q", got)
	}

	req = httptestOrFatal(t, http.MethodGet, "/shows", nil)
	req.SetBasicAuth("admin", "pw")
	rr = serve(s.Handler(), req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	setupTestDB(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, addr, Options{Version: "test"})
	}()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("http://%s/shows", addr))
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s")
	}
}

func httptestOrFatal(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	var bodyReader *os.File
	if body != nil {
		bodyReader = os.Stdin
	}
	_ = bodyReader
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func serve(handler http.Handler, req *http.Request) *httptestResponse {
	rr := &httptestResponse{header: http.Header{}}
	handler.ServeHTTP(rr, req)
	return rr
}

type httptestResponse struct {
	Code    int
	body    []byte
	header  http.Header
	written bool
}

func (r *httptestResponse) Header() http.Header { return r.header }
func (r *httptestResponse) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	if !r.written {
		if r.Code == 0 {
			r.Code = http.StatusOK
		}
		r.written = true
	}
	return len(b), nil
}
func (r *httptestResponse) WriteHeader(code int) { r.Code = code }
