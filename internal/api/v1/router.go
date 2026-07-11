package v1

import (
	_ "embed"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openAPIYAML []byte

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(securityHeadersMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(timeoutMiddleware(60 * time.Second))
	r.Use(jsonOnlyMiddleware)

	if h.Metrics != nil && h.Cfg.MetricsEnabled {
		r.Handle("/metrics", h.Metrics.Handler())
	}

	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		if h.App == nil || !h.App.IsUnlocked() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/openapi.json", h.serveOpenAPI)
		api.Get("/version", h.systemVersion)
		api.Group(func(secured chi.Router) {
			secured.Use(rateLimitMiddleware(h.Cfg.RateLimitRPS, h.Cfg.RateLimitBurst))
			secured.Use(h.authMiddleware)
			secured.Route("/system", func(r chi.Router) { h.mountSystem(r) })
			secured.Route("/operations", func(r chi.Router) { h.mountOperations(r) })
			secured.Route("/tasks", func(r chi.Router) { h.mountTasks(r) })
			secured.Route("/hosts", func(r chi.Router) { h.mountHosts(r) })
			secured.Route("/templates", func(r chi.Router) { h.mountTemplates(r) })
			secured.Route("/backups", func(r chi.Router) { h.mountBackups(r) })
			secured.Route("/destinations", func(r chi.Router) { h.mountDestinations(r) })
			secured.Route("/notifications", func(r chi.Router) { h.mountNotifications(r) })
			secured.Route("/logs", func(r chi.Router) { h.mountLogs(r) })
			secured.Route("/sync", func(r chi.Router) { h.mountSync(r) })
		})
	})

	if strings.TrimSpace(h.Cfg.WebRoot) != "" {
		r.Handle("/*", spaFileServer(h.Cfg.WebRoot))
	}
	return r
}

func (h *Handler) serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	var doc any
	if err := yaml.Unmarshal(openAPIYAML, &doc); err != nil {
		writeError(w, http.StatusInternalServerError, "openapi_error", "failed to load openapi spec")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}
