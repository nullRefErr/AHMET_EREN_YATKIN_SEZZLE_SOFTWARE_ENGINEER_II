// Package rest exposes the calculator over HTTP. It owns request decoding, the error
// envelope and the status codes, and it is the only package that imports a web framework.
package rest

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	"calculator/internal/service"
)

const (
	requestIDHeader = "X-Request-Id"
	requestIDKey    = "request_id"

	defaultMaxBodyBytes int64 = 1 << 20 // 1 MiB
)

// Config holds what the router needs from the outside.
type Config struct {
	// Logger receives the access log. Nil selects slog.Default().
	Logger *slog.Logger
	// AllowedOrigins enables CORS for those origins. Empty means no CORS headers, which
	// is right in production, where nginx serves the app and proxies /api on one origin.
	AllowedOrigins []string
}

// NewRouter builds the HTTP router.
func NewRouter(calculator *service.Calculator, cfg Config) *gin.Engine {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	h := handler{calculator: calculator}

	// gin.New rather than gin.Default: the default engine installs its own text logger,
	// which would sit alongside the structured one instead of replacing it.
	router := gin.New()
	router.Use(
		requestID(), // first, so that every line logged afterwards carries it
		accessLog(cfg.Logger),
		recovery(cfg.Logger),
		limitBody(defaultMaxBodyBytes),
		cors(cfg.AllowedOrigins),
	)

	router.GET("/healthz", h.health)

	v1 := router.Group("/api/v1")
	v1.POST("/calculations", h.createCalculation)
	v1.GET("/operations", h.listOperations)

	return router
}

func requestIDFrom(c *gin.Context) string {
	id, _ := c.Get(requestIDKey)
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// requestID gives every request an identifier, echoes it in a header and puts it in the
// context so that a report from a user can be matched to a log line.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}

// accessLog records one structured line per request. The message is constant and the
// variables are fields, so the lines can be aggregated instead of grepped.
func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.InfoContext(c.Request.Context(), "request completed",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			requestIDKey, requestIDFrom(c),
		)
	}
}

// recovery turns a panic into the same error envelope every other failure uses.
func recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.ErrorContext(c.Request.Context(), "handler panicked",
			"panic", recovered,
			requestIDKey, requestIDFrom(c),
		)
		writeError(c, apiError{http.StatusInternalServerError, codeInternalError, "Something went wrong on our side."})
	})
}

// limitBody stops a large body from being read into memory before it is rejected.
func limitBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// cors answers browser preflights for the configured origins.
func cors(allowed []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && slices.Contains(allowed, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, "+requestIDHeader)
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
