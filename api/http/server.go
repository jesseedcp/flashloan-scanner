package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"

	"github.com/cpchain-network/flashloan-scanner/api/http/handlers"
	apiservice "github.com/cpchain-network/flashloan-scanner/api/service"
	scanws "github.com/cpchain-network/flashloan-scanner/api/ws"
	"github.com/cpchain-network/flashloan-scanner/common/cliapp"
	"github.com/cpchain-network/flashloan-scanner/config"
	"github.com/cpchain-network/flashloan-scanner/database"
)

type Server struct {
	cfg          *config.Config
	db           *database.DB
	jobManager   *apiservice.JobManager
	runnerBridge *apiservice.RunnerBridge
	queryService *apiservice.QueryService
	engine       *gin.Engine
	httpServer   *http.Server
	stopped      atomic.Bool
}

func NewServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	db, err := database.NewDB(ctx, cfg.MasterDb)
	if err != nil {
		return nil, err
	}
	runnerBridge, err := apiservice.NewRunnerBridge(ctx, cfg, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server := &Server{
		cfg:          cfg,
		db:           db,
		jobManager:   apiservice.NewJobManager(),
		runnerBridge: runnerBridge,
	}
	server.queryService = apiservice.NewQueryService(db, server.jobManager, runnerBridge)
	server.engine = server.buildRouter()

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: server.engine,
	}
	return server, nil
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("scan console server stopped unexpectedly", "err", err)
		}
	}()
	log.Info("scan console server started", "addr", s.httpServer.Addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	var result error
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = errors.Join(result, err)
		}
	}
	if s.runnerBridge != nil {
		if err := s.runnerBridge.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	s.stopped.Store(true)
	return result
}

func (s *Server) Stopped() bool {
	return s.stopped.Load()
}

func (s *Server) buildRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	wsHandler := scanws.NewHandler(s.jobManager, s.runnerBridge)
	httpHandler := handlers.New(s.queryService, s.cfg.Scanner.ChainID)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/ws/scan", wsHandler.HandleScan)
	api := router.Group("/api/v1")
	api.GET("/summary", httpHandler.Summary)
	api.GET("/transactions", httpHandler.Transactions)
	api.GET("/transactions/:txHash", httpHandler.TransactionDetail)
	api.GET("/jobs/:jobId/results", httpHandler.JobResults)

	return router
}

var _ cliapp.Lifecycle = (*Server)(nil)
