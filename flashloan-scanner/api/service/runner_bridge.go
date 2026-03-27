package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cpchain-network/flashloan-scanner/config"
	"github.com/cpchain-network/flashloan-scanner/database"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/bootstrap"
	scannerorchestrator "github.com/cpchain-network/flashloan-scanner/scanner/orchestrator"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

type RunnerBridge struct {
	cfg        *config.Config
	db         *database.DB
	rpcClients map[uint64]*rpc.Client

	mu     sync.Mutex
	seeded map[uint64]map[scanner.Protocol]bool
}

func NewRunnerBridge(ctx context.Context, cfg *config.Config, db *database.DB) (*RunnerBridge, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	bridge := &RunnerBridge{
		cfg:        cfg,
		db:         db,
		rpcClients: make(map[uint64]*rpc.Client, len(cfg.RPCs)),
		seeded:     make(map[uint64]map[scanner.Protocol]bool),
	}

	for _, rpcCfg := range cfg.RPCs {
		if rpcCfg == nil {
			continue
		}
		client, err := rpc.DialContext(ctx, rpcCfg.RpcUrl)
		if err != nil {
			_ = bridge.Close()
			return nil, fmt.Errorf("dial rpc for chain %d: %w", rpcCfg.ChainId, err)
		}
		bridge.rpcClients[rpcCfg.ChainId] = client
	}

	return bridge, nil
}

func (b *RunnerBridge) Close() error {
	var result error
	for _, client := range b.rpcClients {
		if client != nil {
			client.Close()
		}
	}
	return result
}

func (b *RunnerBridge) StartJob(ctx context.Context, manager *JobManager, params ScanJobParams) (JobOverview, error) {
	if manager == nil {
		return JobOverview{}, fmt.Errorf("job manager is required")
	}

	overview, err := manager.CreateJob(params)
	if err != nil {
		return JobOverview{}, err
	}
	if err := b.RunJobAsync(ctx, manager, overview.JobID, params); err != nil {
		return JobOverview{}, err
	}

	startedOverview, _ := manager.GetJob(overview.JobID)
	return startedOverview, nil
}

func (b *RunnerBridge) RunJobAsync(ctx context.Context, manager *JobManager, jobID string, params ScanJobParams) error {
	if manager == nil {
		return fmt.Errorf("job manager is required")
	}
	if _, ok := manager.GetJob(jobID); !ok {
		return fmt.Errorf("job %s not found", jobID)
	}
	if err := manager.MarkJobStarted(jobID); err != nil {
		return err
	}

	go func() {
		if err := b.runJob(ctx, manager, jobID, params); err != nil {
			_ = manager.MarkJobFailed(jobID, err.Error())
			return
		}
		_ = manager.MarkJobCompleted(jobID)
	}()
	return nil
}

func (b *RunnerBridge) runJob(ctx context.Context, manager *JobManager, jobID string, params ScanJobParams) error {
	protocols, err := normalizeProtocols(params.Protocols)
	if err != nil {
		return err
	}
	if _, err := b.cfg.RPCByChainID(params.ChainID); err != nil {
		return err
	}

	errCh := make(chan error, len(protocols))
	var wg sync.WaitGroup
	for _, protocol := range protocols {
		protocol := protocol
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.ensureProtocolSeeded(params.ChainID, protocol); err != nil {
				errCh <- fmt.Errorf("%s: %w", protocol, err)
				return
			}
			runner, err := b.buildRunner(params.ChainID, protocol, params.TraceEnabled)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", protocol, err)
				return
			}
			runner.WithSkipTxFetch(b.cfg.Scanner.SkipTxFetch)
			runner.WithObserver(NewJobObserver(jobID, manager))
			if err := runner.RunOnce(ctx, params.ChainID, params.StartBlock, params.EndBlock); err != nil {
				errCh <- fmt.Errorf("%s: %w", protocol, err)
			}
		}()
	}
	wg.Wait()
	close(errCh)

	var result error
	for err := range errCh {
		result = errors.Join(result, err)
	}
	return result
}

func (b *RunnerBridge) buildRunner(chainID uint64, protocol scanner.Protocol, traceEnabled bool) (*scannerorchestrator.ProtocolRunner, error) {
	client, ok := b.rpcClients[chainID]
	if !ok {
		return nil, fmt.Errorf("rpc client not configured for chain %d", chainID)
	}

	loopInterval := time.Duration(b.cfg.Scanner.LoopIntervalSeconds) * time.Second
	switch protocol {
	case scanner.ProtocolAaveV3:
		return scannerorchestrator.BuildAaveV3Runner(b.db, client, chainID, scannerorchestrator.AaveV3RunnerConfig{
			ScannerName:  "aave_v3_scanner",
			LoopInterval: loopInterval,
			BatchSize:    b.cfg.Scanner.BatchSize,
			TraceEnabled: traceEnabled,
		})
	case scanner.ProtocolBalancerV2:
		return scannerorchestrator.BuildBalancerV2Runner(b.db, client, chainID, scannerorchestrator.BalancerV2RunnerConfig{
			ScannerName:  "balancer_v2_scanner",
			LoopInterval: loopInterval,
			BatchSize:    b.cfg.Scanner.BatchSize,
			TraceEnabled: traceEnabled,
		})
	case scanner.ProtocolUniswapV2:
		return scannerorchestrator.BuildUniswapV2Runner(b.db, client, chainID, scannerorchestrator.UniswapV2RunnerConfig{
			ScannerName:  "uniswap_v2_scanner",
			LoopInterval: loopInterval,
			BatchSize:    b.cfg.Scanner.BatchSize,
			TraceEnabled: traceEnabled,
		})
	default:
		return nil, fmt.Errorf("unsupported protocol %s", protocol)
	}
}

func (b *RunnerBridge) TraceProvider(chainID uint64) (scannertrace.Provider, error) {
	client, ok := b.rpcClients[chainID]
	if !ok || client == nil {
		return nil, fmt.Errorf("rpc client not configured for chain %d", chainID)
	}
	return scannertrace.NewGethProvider(client), nil
}

func (b *RunnerBridge) ensureProtocolSeeded(chainID uint64, protocol scanner.Protocol) error {
	b.mu.Lock()
	if b.seeded[chainID] != nil && b.seeded[chainID][protocol] {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	var err error
	switch protocol {
	case scanner.ProtocolAaveV3:
		err = bootstrap.SeedOfficialAaveV3Pools(b.db.ProtocolAddress, chainID, b.cfg.Scanner.Aave.Pools)
	case scanner.ProtocolBalancerV2:
		err = bootstrap.SeedOfficialBalancerV2Vaults(b.db.ProtocolAddress, chainID, b.cfg.Scanner.Balancer.Vaults)
	case scanner.ProtocolUniswapV2:
		if err = bootstrap.SeedOfficialUniswapV2Factories(b.db.ProtocolAddress, chainID, b.cfg.Scanner.UniswapV2.Factories); err != nil {
			return err
		}
		err = bootstrap.SeedOfficialUniswapV2Pairs(b.db.UniswapV2Pair, chainID, b.cfg.Scanner.UniswapV2.Pairs)
	default:
		err = fmt.Errorf("unsupported protocol %s", protocol)
	}
	if err != nil {
		return err
	}

	b.mu.Lock()
	if b.seeded[chainID] == nil {
		b.seeded[chainID] = make(map[scanner.Protocol]bool)
	}
	b.seeded[chainID][protocol] = true
	b.mu.Unlock()
	return nil
}
