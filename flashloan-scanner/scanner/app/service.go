package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cpchain-network/flashloan-scanner/common/cliapp"
	"github.com/cpchain-network/flashloan-scanner/config"
	"github.com/cpchain-network/flashloan-scanner/database"
	"github.com/cpchain-network/flashloan-scanner/scanner/bootstrap"
	scannerorchestrator "github.com/cpchain-network/flashloan-scanner/scanner/orchestrator"
)

const (
	RunModeOnce = "once"
	RunModeLoop = "loop"
)

type FlashloanScannerService struct {
	cfg       *config.Config
	scanCfg   config.Scanner
	db        *database.DB
	rpcClient *rpc.Client
	runner    *scannerorchestrator.ProtocolRunner
	shutdown  context.CancelCauseFunc

	started atomic.Bool
	stopped atomic.Bool

	done chan error
}

func NewFlashloanScannerService(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*FlashloanScannerService, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if !cfg.Scanner.Enabled {
		return nil, errors.New("scanner is disabled in config")
	}
	if cfg.Scanner.ChainID == 0 {
		return nil, errors.New("scanner.chain_id is required")
	}
	if cfg.Scanner.Protocol == "" {
		return nil, errors.New("scanner.protocol is required")
	}

	rpcCfg, err := cfg.RPCByChainID(cfg.Scanner.ChainID)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDB(ctx, cfg.MasterDb)
	if err != nil {
		return nil, err
	}

	rawClient, err := rpc.DialContext(ctx, rpcCfg.RpcUrl)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("dial rpc: %w", err)
	}

	if err := seedProtocolAddresses(db, cfg.Scanner); err != nil {
		rawClient.Close()
		_ = db.Close()
		return nil, err
	}

	runner, err := buildRunner(db, rawClient, cfg.Scanner)
	if err != nil {
		rawClient.Close()
		_ = db.Close()
		return nil, err
	}
	runner.WithSkipTxFetch(cfg.Scanner.SkipTxFetch)

	return &FlashloanScannerService{
		cfg:       cfg,
		scanCfg:   cfg.Scanner,
		db:        db,
		rpcClient: rawClient,
		runner:    runner,
		shutdown:  shutdown,
		done:      make(chan error, 1),
	}, nil
}

func (s *FlashloanScannerService) Start(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return errors.New("scanner service already started")
	}

	go func() {
		var err error
		switch normalizeRunMode(s.scanCfg.RunMode) {
		case RunModeOnce:
			err = s.runOnceMode(ctx)
		case RunModeLoop:
			err = s.runner.RunLoop(ctx, s.scanCfg.ChainID)
		default:
			err = fmt.Errorf("unsupported scanner run mode: %s", s.scanCfg.RunMode)
		}

		s.done <- err
		close(s.done)

		if err == nil && normalizeRunMode(s.scanCfg.RunMode) == RunModeOnce {
			s.shutdown(nil)
			return
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			s.shutdown(err)
		}
	}()

	log.Info("flashloan scanner started", "protocol", s.scanCfg.Protocol, "chainId", s.scanCfg.ChainID, "mode", normalizeRunMode(s.scanCfg.RunMode))
	return nil
}

func (s *FlashloanScannerService) Stop(ctx context.Context) error {
	var result error

	if s.started.Load() && !s.stopped.Load() {
		select {
		case err, ok := <-s.done:
			if ok && err != nil && !errors.Is(err, context.Canceled) {
				result = errors.Join(result, err)
			}
		case <-ctx.Done():
			result = errors.Join(result, ctx.Err())
		}
	}

	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}

	s.stopped.Store(true)
	return result
}

func (s *FlashloanScannerService) Stopped() bool {
	return s.stopped.Load()
}

func (s *FlashloanScannerService) runOnceMode(ctx context.Context) error {
	startBlock := s.scanCfg.StartBlock
	endBlock := s.scanCfg.EndBlock
	if endBlock == 0 {
		client := ethclient.NewClient(s.rpcClient)
		latest, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		endBlock = latest
	}
	if endBlock < startBlock {
		return fmt.Errorf("scanner.end_block (%d) is less than scanner.start_block (%d)", endBlock, startBlock)
	}
	return s.runner.RunOnce(ctx, s.scanCfg.ChainID, startBlock, endBlock)
}

func seedProtocolAddresses(db *database.DB, cfg config.Scanner) error {
	switch cfg.Protocol {
	case "aave_v3":
		return bootstrap.SeedOfficialAaveV3Pools(db.ProtocolAddress, cfg.ChainID, cfg.Aave.Pools)
	case "balancer_v2":
		return bootstrap.SeedOfficialBalancerV2Vaults(db.ProtocolAddress, cfg.ChainID, cfg.Balancer.Vaults)
	case "uniswap_v2":
		if err := bootstrap.SeedOfficialUniswapV2Factories(db.ProtocolAddress, cfg.ChainID, cfg.UniswapV2.Factories); err != nil {
			return err
		}
		return bootstrap.SeedOfficialUniswapV2Pairs(db.UniswapV2Pair, cfg.ChainID, cfg.UniswapV2.Pairs)
	default:
		return fmt.Errorf("unsupported scanner protocol for seed: %s", cfg.Protocol)
	}
}

func buildRunner(db *database.DB, rawClient *rpc.Client, cfg config.Scanner) (*scannerorchestrator.ProtocolRunner, error) {
	switch cfg.Protocol {
	case "aave_v3":
		return scannerorchestrator.BuildAaveV3Runner(db, rawClient, cfg.ChainID, scannerorchestrator.AaveV3RunnerConfig{
			ScannerName:  "aave_v3_scanner",
			LoopInterval: time.Duration(cfg.LoopIntervalSeconds) * time.Second,
			BatchSize:    cfg.BatchSize,
			TraceEnabled: cfg.TraceEnabled,
		})
	case "balancer_v2":
		return scannerorchestrator.BuildBalancerV2Runner(db, rawClient, cfg.ChainID, scannerorchestrator.BalancerV2RunnerConfig{
			ScannerName:  "balancer_v2_scanner",
			LoopInterval: time.Duration(cfg.LoopIntervalSeconds) * time.Second,
			BatchSize:    cfg.BatchSize,
			TraceEnabled: cfg.TraceEnabled,
		})
	case "uniswap_v2":
		return scannerorchestrator.BuildUniswapV2Runner(db, rawClient, cfg.ChainID, scannerorchestrator.UniswapV2RunnerConfig{
			ScannerName:  "uniswap_v2_scanner",
			LoopInterval: time.Duration(cfg.LoopIntervalSeconds) * time.Second,
			BatchSize:    cfg.BatchSize,
			TraceEnabled: cfg.TraceEnabled,
		})
	default:
		return nil, fmt.Errorf("unsupported scanner protocol: %s", cfg.Protocol)
	}
}

func normalizeRunMode(raw string) string {
	switch raw {
	case "", RunModeLoop:
		return RunModeLoop
	case RunModeOnce:
		return RunModeOnce
	default:
		return raw
	}
}

var _ cliapp.Lifecycle = (*FlashloanScannerService)(nil)
