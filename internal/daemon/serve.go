package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apiv1 "dback/internal/api/v1"
	"dback/internal/config"
	"dback/internal/controlplane"
	"dback/internal/notify"
	"dback/internal/trigger"
)

type Server struct {
	cfg      config.Config
	cp       *controlplane.Service
	triggers *trigger.Registry
	notify   *notify.Router
	audit    func()
	metrics  func()
	api      *apiv1.Handler
	http     *http.Server
	ready    bool
	unlock   func()
}

func NewServer(cfg config.Config, cp *controlplane.Service, triggers *trigger.Registry, notifyRouter *notify.Router, apiHandler *apiv1.Handler, unlock func()) *Server {
	if apiHandler == nil {
		apiHandler = &apiv1.Handler{
			App:      cp.App,
			CP:       cp,
			Triggers: triggers,
			Notify:   notifyRouter,
			Cfg:      cfg,
		}
	}
	s := &Server{cfg: cfg, cp: cp, triggers: triggers, notify: notifyRouter, api: apiHandler}
	s.http = &http.Server{
		Addr:              cfg.Listen,
		Handler:           apiHandler.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.unlock = unlock
	return s
}

func (s *Server) SetShutdownHooks(auditStop, metricsStop func()) {
	s.audit = auditStop
	s.metrics = metricsStop
}

func (s *Server) Run(ctx context.Context) error {
	release, err := AcquireLock(s.cfg.LockPath())
	if err != nil {
		return err
	}
	defer release()

	s.ready = true
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.http.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case sig := <-sigCh:
		shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		if s.triggers != nil {
			_ = s.triggers.Stop(shutdownCtx)
		}
		if s.notify != nil {
			s.notify.Stop()
		}
		if s.audit != nil {
			s.audit()
		}
		if s.metrics != nil {
			s.metrics()
		}
		_ = s.cp.Dispatcher.Shutdown(shutdownCtx)
		if s.unlock != nil {
			s.unlock()
		}
		if err := <-errCh; err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("shutdown after %v: %w", sig, err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		if s.triggers != nil {
			_ = s.triggers.Stop(shutdownCtx)
		}
		if s.notify != nil {
			s.notify.Stop()
		}
		if s.audit != nil {
			s.audit()
		}
		if s.metrics != nil {
			s.metrics()
		}
		_ = s.cp.Dispatcher.Shutdown(shutdownCtx)
		if s.unlock != nil {
			s.unlock()
		}
		return ctx.Err()
	}
}
