package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/bgengine/pkg/engine"
)

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Host           string        // Host to bind to (default "localhost")
	Port           int           // Port to listen on (default 8080)
	ReadTimeout    time.Duration // Read timeout (default 30s)
	WriteTimeout   time.Duration // Write timeout (default 30s)
	IdleTimeout    time.Duration // Idle timeout (default 60s)
	MaxFastWorkers int           // Max concurrent fast operations (default 100)
	MaxSlowWorkers int           // Max concurrent slow operations (default 4)
}

// DefaultConfig returns a ServerConfig with sensible defaults.
func DefaultConfig() ServerConfig {
	return ServerConfig{
		Host:           "localhost",
		Port:           8080,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxFastWorkers: 100,
		MaxSlowWorkers: 4,
	}
}

// Server is the HTTP API server.
type Server struct {
	config   ServerConfig
	engine   *engine.Engine
	handlers *Handlers
	server   *http.Server
	pool     *WorkerPool
	version  string
}

// NewServer creates a new API server.
func NewServer(e *engine.Engine, config ServerConfig, version string) *Server {
	poolConfig := PoolConfig{
		MaxFastWorkers: config.MaxFastWorkers,
		MaxSlowWorkers: config.MaxSlowWorkers,
	}
	if poolConfig.MaxFastWorkers <= 0 {
		poolConfig.MaxFastWorkers = 100
	}
	if poolConfig.MaxSlowWorkers <= 0 {
		poolConfig.MaxSlowWorkers = 4
	}

	pool := NewWorkerPool(poolConfig)
	handlers := NewHandlersWithPool(e, version, pool)

	return &Server{
		config:   config,
		engine:   e,
		handlers: handlers,
		pool:     pool,
		version:  version,
	}
}

// Pool returns the worker pool for monitoring.
func (s *Server) Pool() *WorkerPool {
	return s.pool
}

// corsMiddleware adds CORS headers for browser access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", s.handlers.Health)
	mux.HandleFunc("POST /api/evaluate", s.handlers.Evaluate)
	mux.HandleFunc("POST /api/move", s.handlers.Move)
	mux.HandleFunc("POST /api/cube", s.handlers.Cube)
	mux.HandleFunc("POST /api/rollout", s.handlers.Rollout)
	mux.HandleFunc("GET /api/rollout/stream", s.handlers.RolloutSSE)
	mux.HandleFunc("/api/ws", s.handlers.WebSocket)
	mux.HandleFunc("POST /api/fibsboard", s.handlers.HandleFIBSBoard)

	// Tutor API routes
	mux.HandleFunc("POST /api/tutor/move", s.handlers.HandleTutorMove)
	mux.HandleFunc("POST /api/tutor/cube", s.handlers.HandleTutorCube)
	mux.HandleFunc("POST /api/tutor/game", s.handlers.HandleAnalyzeGame)

	// Also allow GET for health with legacy pattern
	mux.HandleFunc("/api/health", s.handlers.Health)

	// Apply middleware
	handler := corsMiddleware(loggingMiddleware(mux))

	return handler
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.setupRoutes(),
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	log.Printf("Starting GoBG API server v%s on %s", s.version, addr)
	log.Printf("Endpoints:")
	log.Printf("  GET  /api/health      - Health check")
	log.Printf("  POST /api/evaluate    - Evaluate position")
	log.Printf("  POST /api/move        - Find best moves")
	log.Printf("  POST /api/cube        - Cube decision")
	log.Printf("  POST /api/rollout     - Monte Carlo rollout")
	log.Printf("  POST /api/fibsboard   - Analyze FIBS board string")
	log.Printf("  POST /api/tutor/move  - Analyze played move")
	log.Printf("  POST /api/tutor/cube  - Analyze cube decision")
	log.Printf("  POST /api/tutor/game  - Analyze complete game")
	log.Printf("  WS   /api/ws          - WebSocket for real-time analysis")

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// ListenAndServeWithGracefulShutdown starts the server and handles shutdown signals.
func (s *Server) ListenAndServeWithGracefulShutdown() error {
	// Channel to listen for errors from server
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal or error
	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("Received signal %v, shutting down...", sig)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
