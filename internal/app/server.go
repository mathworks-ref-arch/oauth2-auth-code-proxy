// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/admin"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/docs"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/mwlegacy"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/redirect"
	v1 "github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/v1"
	"go.uber.org/zap"
)

type Server struct {
	oauth     domain.OAuthService
	apps      map[string]domain.AppConfig
	logger    *zap.Logger
	health    *admin.HealthChecker
	listenCfg domain.ListenConfig
}

func NewServer(svc domain.OAuthService, apps map[string]domain.AppConfig, info domain.BuildInfo, logger *zap.Logger) *Server {
	hc := admin.NewHealthChecker(apps, info, logger.Named("health"))
	hc.InitURLs()
	return &Server{
		oauth:  svc,
		apps:   apps,
		logger: logger,
		health: hc,
	}
}

func (s *Server) RESTHandler() http.Handler {
	mux := http.NewServeMux()
	v1.Register(mux, s.oauth, s.apps, s.logger.Named("v1"))
	admin.Register(mux, "/v1", s.health)
	redirect.Register(mux, "/v1", s.apps)
	docs.Register(mux, "/v1", s.listenCfg)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	return s.middleware(mux)
}

func (s *Server) MWLegacyHandler() http.Handler {
	mux := http.NewServeMux()
	mwlegacy.Register(mux, s.oauth, s.apps, s.logger.Named("mwlegacy"))
	admin.Register(mux, "/oauth/v1", s.health)
	redirect.Register(mux, "/oauth/v1", s.apps)
	docs.Register(mux, "/oauth/v1", s.listenCfg)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	return s.middleware(mux)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if !s.health.Loaded() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready")) //nolint:errcheck
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		defer func() {
			if rv := recover(); rv != nil {
				s.logger.Error("panic recovered", zap.Any("error", rv))
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		s.logger.Debug("request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path))

		next.ServeHTTP(w, r)
	})
}

func (s *Server) ListenAndServe(ctx context.Context, cfg domain.ListenConfig) error {
	s.listenCfg = cfg
	errCh := make(chan error, 2)

	restSrv := s.startHTTP("REST API", cfg.RESTPort, s.RESTHandler(), cfg, errCh)

	var legacySrv *http.Server
	if cfg.MWLegacyEnabled {
		legacySrv = s.startHTTP("legacy API", cfg.MWLegacyPort, s.MWLegacyHandler(), cfg, errCh)
	}

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	select {
	case err := <-errCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.shutdownAll(shutdownCtx, restSrv, legacySrv) //nolint:errcheck
		return fmt.Errorf("server error: %w", err)
	case <-sigCtx.Done():
		s.logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.shutdownAll(shutdownCtx, restSrv, legacySrv)
	}
}

func (s *Server) startHTTP(name string, port string, handler http.Handler, cfg domain.ListenConfig, errCh chan<- error) *http.Server {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			s.logger.Info(name+" starting with TLS", zap.String("port", port))
			if err := srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		} else {
			s.logger.Info(name+" starting", zap.String("port", port))
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}
	}()
	return srv
}

func (s *Server) shutdownAll(ctx context.Context, servers ...*http.Server) error {
	for _, srv := range servers {
		if srv == nil {
			continue
		}
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}
	return nil
}
