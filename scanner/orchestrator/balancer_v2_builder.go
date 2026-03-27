package orchestrator

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cpchain-network/flashloan-scanner/database"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/aggregator"
	scannercursor "github.com/cpchain-network/flashloan-scanner/scanner/cursor"
	scannerextractor "github.com/cpchain-network/flashloan-scanner/scanner/extractor"
	scannerfetcher "github.com/cpchain-network/flashloan-scanner/scanner/fetcher"
	scannerregistry "github.com/cpchain-network/flashloan-scanner/scanner/registry"
	scannerstore "github.com/cpchain-network/flashloan-scanner/scanner/store"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
	scannerverifier "github.com/cpchain-network/flashloan-scanner/scanner/verifier"
)

type BalancerV2RunnerConfig struct {
	ScannerName  string
	LoopInterval time.Duration
	BatchSize    uint64
	TraceEnabled bool
}

func BuildBalancerV2Runner(
	db *database.DB,
	rawClient *rpc.Client,
	chainID uint64,
	cfg BalancerV2RunnerConfig,
) (*ProtocolRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if rawClient == nil {
		return nil, fmt.Errorf("rpc client is required")
	}
	if cfg.ScannerName == "" {
		cfg.ScannerName = "balancer_v2_scanner"
	}
	if cfg.LoopInterval <= 0 {
		cfg.LoopInterval = 5 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 500
	}

	reg, err := scannerregistry.NewMemoryRegistryFromDB(chainID, db.ProtocolAddress, db.UniswapV2Pair)
	if err != nil {
		return nil, err
	}

	store := scannerstore.NewGormStore(db)
	txFetcher := scannerfetcher.NewEthereumTxFetcher(rawClient, db.ContractEvents, store)
	extractor, err := scannerextractor.NewBalancerV2CandidateExtractor(reg, db.ContractEvents)
	if err != nil {
		return nil, err
	}
	verifier, err := scannerverifier.NewBalancerV2Verifier(db.ContractEvents, db.ObservedTx, db.InteractionAssetLeg)
	if err != nil {
		return nil, err
	}
	var interactionVerifier scannerverifier.InteractionVerifier = verifier
	if cfg.TraceEnabled {
		traceVerifier, err := scannerverifier.NewBalancerV2TraceVerifier(verifier, scannertrace.NewGethProvider(rawClient))
		if err != nil {
			return nil, err
		}
		interactionVerifier = traceVerifier
	}
	txAggregator := aggregator.NewSimpleTxAggregator(db.ProtocolInteraction, db.FlashloanTx)
	cursorManager := scannercursor.NewGormManager(db.ScannerCursor)

	runner, err := NewProtocolRunner(
		cfg.ScannerName,
		scanner.ProtocolBalancerV2,
		txFetcher,
		store,
		store,
		store,
		extractor,
		interactionVerifier,
		txAggregator,
	)
	if err != nil {
		return nil, err
	}

	return runner.
		WithLoopInterval(cfg.LoopInterval).
		WithBatchSize(cfg.BatchSize).
		WithCursorManager(cursorManager).
		WithLatestBlockProvider(txFetcher), nil
}
