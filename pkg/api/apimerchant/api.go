package apimerchant

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gustavooferreira/pgw-payment-gateway-service/pkg/api"
	"github.com/gustavooferreira/pgw-payment-gateway-service/pkg/api/middleware"
	"github.com/gustavooferreira/pgw-payment-gateway-service/pkg/core"
	"github.com/gustavooferreira/pgw-payment-gateway-service/pkg/core/log"
)

// Server is the webserver environment, which holds all its dependencies.
type Server struct {
	Logger     log.Logger
	Repo       core.Repository
	PProcessor core.PaymentProcessor

	AuthServiceHost string
	AuthServicePort int

	Router     *gin.Engine
	HTTPServer http.Server
	HTTPClient *http.Client
}

// NewServer creates a new server.
func NewServer(addr string, port int, devMode bool, authServiceHost string, authServicePort int,
	logger log.Logger, httpClient *http.Client, repo core.Repository, pproc core.PaymentProcessor) *Server {
	s := &Server{Logger: logger, Repo: repo, HTTPClient: httpClient,
		AuthServiceHost: authServiceHost, AuthServicePort: authServicePort,
		PProcessor: pproc}

	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}

	s.Router = gin.New()

	s.Router.Use(
		middleware.GinReqLogger(logger, time.RFC3339, "request served by merchant API", "http-router-mux"),
	)
	if !devMode {
		s.Router.Use(gin.Recovery())
	}

	// Create http.Server
	s.HTTPServer = http.Server{
		Addr:           fmt.Sprintf("%s:%d", addr, port),
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.setupRoutes(devMode)

	return s
}

// setupRoutes creates routes for all handlers
func (s *Server) setupRoutes(devMode bool) {
	s.Router.NoRoute(api.NoRoute)
	v1 := s.Router.Group("/api/v1")

	basicAuthMW := middleware.GinBasicAuth(s.Logger, s.HTTPClient, s.AuthServiceHost, s.AuthServicePort)

	v1.POST("/authorise", basicAuthMW, s.AuthoriseTransaction)
	v1.POST("/capture", basicAuthMW, s.CaptureTransaction)
	v1.POST("/refund", basicAuthMW, s.RefundTransaction)
	v1.POST("/void", basicAuthMW, s.VoidTransaction)

}

// ListenAndServe listens and serves incoming requests.
func (s *Server) ListenAndServe() error {
	if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ShutDown gracefully shuts down server.
func (s *Server) ShutDown(ctx context.Context) error {
	return s.HTTPServer.Shutdown(ctx)
}
