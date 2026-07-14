package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpServer "github.com/mark3labs/mcp-go/server"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/translator"
)

type Server struct {
	cfg *config.Config
	svc *translator.Service
	mcp *mcpServer.MCPServer
}

func New(cfg *config.Config) (*Server, error) {
	svc, err := translator.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("init translator: %w", err)
	}
	s := &Server{
		cfg: cfg,
		svc: svc,
		mcp: mcpServer.NewMCPServer(
			"translate-mcp",
			"1.0.0",
			mcpServer.WithToolCapabilities(false),
		),
	}
	s.registerTools()
	return s, nil
}

func (s *Server) Close() error {
	return s.svc.Close()
}

func (s *Server) Run() error {
	if s.cfg.Server.Transport == "stdio" {
		return mcpServer.ServeStdio(s.mcp)
	}
	return s.runHTTP()
}

func (s *Server) runHTTP() error {
	httpServer := mcpServer.NewStreamableHTTPServer(
		s.mcp,
		mcpServer.WithEndpointPath("/mcp"),
		mcpServer.WithDisableLocalhostProtection(true),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler())
	mux.Handle(s.cfg.Metrics.Path, s.svc.Metrics().Handler())
	mux.Handle("/mcp", httpServer)

	handler := withCORS(mux, s.cfg.Server.CORS)
	handler = withAuth(handler, s.cfg.Server.Auth)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	slog.Info("starting http server", "addr", addr, "transport", "streamable-http")
	return http.ListenAndServe(addr, handler)
}

func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		providers := s.svc.Health(ctx)
		resp := map[string]any{
			"status":    "ok",
			"version":   "1.0.0",
			"providers": providers,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultJSON(v)
}

func errorResult(err error) *mcp.CallToolResult {
	return mcp.NewToolResultError(err.Error())
}

func withAuth(next http.Handler, auth config.AuthConfig) http.Handler {
	if auth.Type != "bearer" || auth.Token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) || header[len(prefix):] != auth.Token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler, cors config.CORSConfig) http.Handler {
	if len(cors.AllowedOrigins) == 0 {
		return next
	}
	allowAll := len(cors.AllowedOrigins) == 1 && cors.AllowedOrigins[0] == "*"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := allowAll
		if !allowed {
			for _, o := range cors.AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
		}
		if allowed {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
