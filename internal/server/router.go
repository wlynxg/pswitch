package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pswitch/internal/adminui"
	"pswitch/internal/config"
	"pswitch/internal/pool"
	"pswitch/internal/protocol/anthropic"
	"pswitch/internal/protocol/openai"
	pruntime "pswitch/internal/runtime"
)

const AdminPrefix = "/dashboard"

const (
	defaultUpstreamDialTimeout           = 3 * time.Second
	defaultUpstreamKeepAlive             = 30 * time.Second
	defaultUpstreamTLSHandshakeTimeout   = 3 * time.Second
	defaultUpstreamResponseHeaderTimeout = 12 * time.Second
	defaultUpstreamIdleConnTimeout       = 90 * time.Second
	defaultUpstreamMaxIdleConns          = 128
	defaultUpstreamMaxIdleConnsPerHost   = 32
)

func NewRouter(manager *pruntime.Manager, adminToken string) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	adminHandler := adminui.New(manager, adminToken)
	router.Handle(AdminPrefix, http.RedirectHandler(AdminPrefix+"/", http.StatusPermanentRedirect))
	router.Handle(AdminPrefix+"/", http.StripPrefix(AdminPrefix, adminHandler))
	router.Handle(AdminPrefix+"/*", http.StripPrefix(AdminPrefix, adminHandler))
	router.Handle("/*", &dispatcher{
		manager:        manager,
		upstreamClient: newUpstreamClient(),
	})

	return router
}

type dispatcher struct {
	manager        *pruntime.Manager
	upstreamClient *http.Client
}

func (d *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, providerPool := d.manager.Snapshot()
	route, ok := matchRoute(cfg.Routes, r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	openAIHandler := openai.NewHandler(providerPool, openai.Options{
		Client:        d.upstreamClient,
		Mode:          pool.Mode(cfg.Mode),
		Metrics:       d.manager.Metrics(),
		ProviderNames: route.Providers,
	})

	var handler http.Handler
	switch route.Kind {
	case "openai":
		handler = openAIHandler
	case "anthropic":
		handler = anthropic.NewHandler(anthropic.Options{
			Model:         route.Model,
			UpstreamModel: route.UpstreamModel,
			Upstream:      openAIHandler,
		})
	default:
		http.NotFound(w, r)
		return
	}

	if route.Prefix != "/" {
		handler = http.StripPrefix(route.Prefix, handler)
	}
	handler.ServeHTTP(w, r)
}

func matchRoute(routes []config.Route, requestPath string) (config.Route, bool) {
	var (
		match    config.Route
		matchLen = -1
	)
	for _, route := range routes {
		if !pathMatches(route.Prefix, requestPath) {
			continue
		}
		if len(route.Prefix) > matchLen {
			match = route
			matchLen = len(route.Prefix)
		}
	}
	return match, matchLen >= 0
}

func pathMatches(prefix, requestPath string) bool {
	if prefix == "/" {
		return true
	}
	return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
}

func newUpstreamClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   defaultUpstreamDialTimeout,
				KeepAlive: defaultUpstreamKeepAlive,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          defaultUpstreamMaxIdleConns,
			MaxIdleConnsPerHost:   defaultUpstreamMaxIdleConnsPerHost,
			IdleConnTimeout:       defaultUpstreamIdleConnTimeout,
			TLSHandshakeTimeout:   defaultUpstreamTLSHandshakeTimeout,
			ExpectContinueTimeout: time.Second,
			ResponseHeaderTimeout: defaultUpstreamResponseHeaderTimeout,
		},
	}
}
