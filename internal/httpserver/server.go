package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

type Server struct {
	listener net.Listener
	server   *http.Server
}

func New(addr string, metricsHandler http.Handler) (*Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{listener: listener, server: &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}}, nil
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		errCh <- s.server.Shutdown(shutdownCtx)
	}()

	if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-errCh
}
