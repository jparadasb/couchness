package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/highercomve/couchness/web/handlers"
	"github.com/highercomve/couchness/web/jobs"
	"github.com/highercomve/couchness/web/repository"
	"github.com/highercomve/couchness/web/services"
	"github.com/highercomve/couchness/web/templates"
)

// DefaultAddress is the default listen address (8080 and 3000 are taken on the target host).
const DefaultAddress = ":8085"

// Options configures the web server.
type Options struct {
	Auth    string // "user:password" enables HTTP basic auth; empty disables it
	Version string // shown in the footer
}

// Server wires repositories, services, jobs, templates and handlers.
type Server struct {
	handler http.Handler
	options Options
}

// New builds the server. Errors if Auth is non-empty but lacks ":" or if templates fail to parse.
func New(options Options) (*Server, error) {
	if options.Auth != "" {
		if !stringsContains(options.Auth, ":") {
			return nil, errors.New("auth must be in the form user:password")
		}
	}

	config := repository.NewConfig()
	runner := jobs.NewRunner(20)
	renderer, err := templates.New()
	if err != nil {
		return nil, err
	}

	h := handlers.New(handlers.Deps{
		Library:  services.NewLibrary(config),
		Shows:    services.NewShows(repository.NewShows()),
		Movies:   services.NewMovies(repository.NewMovies(), config),
		Config:   config,
		Jobs:     runner,
		Renderer: renderer,
		Version:  options.Version,
	})

	mux := http.NewServeMux()
	h.Register(mux)

	var handler http.Handler = mux
	handler = handlers.BasicAuth(handler, options.Auth)
	handler = logging(handler)

	return &Server{handler: handler, options: options}, nil
}

// Handler returns the root http.Handler (logging + optional basic auth + mux).
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Run serves on addr until ctx is done, then shuts down gracefully (10s timeout).
// Logs "couchness web listening on <addr>". Returns nil when closed via ctx.
func Run(ctx context.Context, addr string, options Options) error {
	s, err := New(options)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("couchness web listening on %s", addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rr.status, time.Since(start))
	})
}
