package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/cpchain-network/flashloan-scanner/scanner"
	scanneraggregator "github.com/cpchain-network/flashloan-scanner/scanner/aggregator"
	scannercursor "github.com/cpchain-network/flashloan-scanner/scanner/cursor"
	scannerextractor "github.com/cpchain-network/flashloan-scanner/scanner/extractor"
	scannerfetcher "github.com/cpchain-network/flashloan-scanner/scanner/fetcher"
	scannerstore "github.com/cpchain-network/flashloan-scanner/scanner/store"
	scannerverifier "github.com/cpchain-network/flashloan-scanner/scanner/verifier"
)

type ProtocolRunner struct {
	scannerName      string
	protocol         scanner.Protocol
	fetcher          scannerfetcher.TxFetcher
	txStore          scannerstore.TransactionStore
	interactionStore scannerstore.InteractionStore
	legStore         scannerstore.LegStore
	extractor        scannerextractor.CandidateExtractor
	verifier         scannerverifier.InteractionVerifier
	aggregator       scanneraggregator.TxAggregator
	cursorManager    scannercursor.Manager
	blockProvider    scannerfetcher.LatestBlockProvider
	observer         ProtocolRunObserver
	loopInterval     time.Duration
	batchSize        uint64
	skipTxFetch      bool
}

func NewProtocolRunner(
	scannerName string,
	protocol scanner.Protocol,
	fetcher scannerfetcher.TxFetcher,
	txStore scannerstore.TransactionStore,
	interactionStore scannerstore.InteractionStore,
	legStore scannerstore.LegStore,
	extractor scannerextractor.CandidateExtractor,
	verifier scannerverifier.InteractionVerifier,
	aggregator scanneraggregator.TxAggregator,
) (*ProtocolRunner, error) {
	if scannerName == "" {
		return nil, errors.New("scanner name required")
	}
	if extractor == nil || verifier == nil || fetcher == nil || txStore == nil || interactionStore == nil || legStore == nil || aggregator == nil {
		return nil, errors.New("runner dependency missing")
	}
	if extractor.Protocol() != protocol {
		return nil, fmt.Errorf("extractor protocol mismatch: %s", extractor.Protocol())
	}
	if verifier.Protocol() != protocol {
		return nil, fmt.Errorf("verifier protocol mismatch: %s", verifier.Protocol())
	}
	return &ProtocolRunner{
		scannerName:      scannerName,
		protocol:         protocol,
		fetcher:          fetcher,
		txStore:          txStore,
		interactionStore: interactionStore,
		legStore:         legStore,
		extractor:        extractor,
		verifier:         verifier,
		aggregator:       aggregator,
		loopInterval:     5 * time.Second,
		batchSize:        500,
	}, nil
}

func (r *ProtocolRunner) WithLoopInterval(interval time.Duration) *ProtocolRunner {
	if interval > 0 {
		r.loopInterval = interval
	}
	return r
}

func (r *ProtocolRunner) WithBatchSize(batchSize uint64) *ProtocolRunner {
	if batchSize > 0 {
		r.batchSize = batchSize
	}
	return r
}

func (r *ProtocolRunner) WithCursorManager(manager scannercursor.Manager) *ProtocolRunner {
	r.cursorManager = manager
	return r
}

func (r *ProtocolRunner) WithSkipTxFetch(skip bool) *ProtocolRunner {
	r.skipTxFetch = skip
	return r
}

func (r *ProtocolRunner) WithLatestBlockProvider(provider scannerfetcher.LatestBlockProvider) *ProtocolRunner {
	r.blockProvider = provider
	return r
}

func (r *ProtocolRunner) WithObserver(observer ProtocolRunObserver) *ProtocolRunner {
	r.observer = observer
	return r
}

func (r *ProtocolRunner) RunOnce(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) (err error) {
	stats := protocolRunStats{}
	progressEmitted := false
	lastProgressBlock := uint64(0)
	r.notifyProtocolStarted(chainID, fromBlock, toBlock)
	defer func() {
		if err != nil {
			r.notifyProtocolFailed(chainID, fromBlock, toBlock, stats.currentBlock, err)
			return
		}
		needsFinalProgress := !progressEmitted || lastProgressBlock < toBlock
		if stats.currentBlock < toBlock {
			stats.currentBlock = toBlock
		}
		if needsFinalProgress {
			r.notifyProtocolProgress(chainID, fromBlock, toBlock, stats)
		}
		r.notifyProtocolCompleted(chainID, fromBlock, toBlock, stats)
	}()

	if !r.skipTxFetch {
		if err = r.fetcher.FetchRange(ctx, chainID, fromBlock, toBlock); err != nil {
			return err
		}
	}

	txs, err := r.txStore.ListObservedTransactionsByBlockRange(ctx, chainID, fromBlock, toBlock)
	if err != nil {
		return err
	}
	stats.observedTransactions = len(txs)
	if len(txs) == 0 {
		return nil
	}

	interactions, legs, err := r.extractor.Extract(ctx, chainID, txs)
	if err != nil {
		return err
	}
	stats.candidateInteractions = len(interactions)
	if len(interactions) == 0 {
		return nil
	}

	if err := r.interactionStore.UpsertInteractions(ctx, interactions); err != nil {
		return err
	}
	for interactionID, interactionLegs := range groupLegsByInteractionID(legs) {
		if err := r.legStore.ReplaceInteractionLegs(ctx, interactionID, interactionLegs); err != nil {
			return err
		}
	}

	seenTxs := make(map[string]struct{})
	orderedTxHashes := make([]string, 0, len(interactions))
	for _, interaction := range interactions {
		stats.advanceCurrentBlock(parseUint64(interaction.BlockNumber))
		result, verifiedLegs, err := r.verifier.Verify(ctx, chainID, interaction)
		if err != nil {
			return err
		}
		if result != nil {
			if err := r.interactionStore.UpdateVerificationResult(ctx, *result); err != nil {
				return err
			}
		}
		if verifiedLegs != nil {
			if err := r.legStore.ReplaceInteractionLegs(ctx, interaction.InteractionID, verifiedLegs); err != nil {
				return err
			}
		}
		if result != nil {
			if result.Verified {
				stats.verifiedInteractions++
			}
			if result.Strict {
				stats.strictInteractions++
			}
		}
		if _, ok := seenTxs[interaction.TxHash]; ok {
			continue
		}
		seenTxs[interaction.TxHash] = struct{}{}
		orderedTxHashes = append(orderedTxHashes, interaction.TxHash)
	}

	for _, txHash := range orderedTxHashes {
		summary, err := r.aggregator.AggregateByTx(ctx, chainID, txHash)
		if err != nil {
			return err
		}
		if summary == nil {
			continue
		}
		stats.findings++
		if summary.ContainsVerifiedInteraction {
			stats.verifiedFindings++
		}
		if summary.ContainsVerifiedStrictInteraction {
			stats.strictFindings++
		}
		stats.advanceCurrentBlock(parseUint64(summary.BlockNumber))
		r.notifyFinding(chainID, *summary)
		r.notifyProtocolProgress(chainID, fromBlock, toBlock, stats)
		progressEmitted = true
		lastProgressBlock = stats.currentBlock
	}
	return nil
}

func (r *ProtocolRunner) RunLoop(ctx context.Context, chainID uint64) error {
	if r.cursorManager == nil || r.blockProvider == nil {
		return errors.New("run loop requires both cursor manager and latest block provider")
	}
	if r.batchSize == 0 {
		return errors.New("batch size must be greater than zero")
	}

	cursorType := string(scanner.CursorTypeVerification)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		latestBlock, err := r.blockProvider.LatestBlockNumber(ctx)
		if err != nil {
			return err
		}
		lastProcessed, err := r.cursorManager.Get(ctx, r.scannerName, chainID, cursorType)
		if err != nil {
			return err
		}
		nextBlock := lastProcessed
		if nextBlock > 0 {
			nextBlock++
		}
		if nextBlock > latestBlock {
			if err := sleepWithContext(ctx, r.loopInterval); err != nil {
				return err
			}
			continue
		}

		toBlock := nextBlock + r.batchSize - 1
		if toBlock > latestBlock {
			toBlock = latestBlock
		}
		if err := r.RunOnce(ctx, chainID, nextBlock, toBlock); err != nil {
			return err
		}
		if err := r.cursorManager.Save(ctx, r.scannerName, chainID, cursorType, toBlock); err != nil {
			return err
		}
		if toBlock >= latestBlock {
			if err := sleepWithContext(ctx, r.loopInterval); err != nil {
				return err
			}
		}
	}
}

func groupLegsByInteractionID(legs []scanner.InteractionLeg) map[string][]scanner.InteractionLeg {
	grouped := make(map[string][]scanner.InteractionLeg)
	for _, leg := range legs {
		grouped[leg.InteractionID] = append(grouped[leg.InteractionID], leg)
	}
	return grouped
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type protocolRunStats struct {
	currentBlock          uint64
	observedTransactions  int
	candidateInteractions int
	verifiedInteractions  int
	strictInteractions    int
	findings              int
	verifiedFindings      int
	strictFindings        int
}

func (s *protocolRunStats) advanceCurrentBlock(block uint64) {
	if block > s.currentBlock {
		s.currentBlock = block
	}
}

func (r *ProtocolRunner) notifyProtocolStarted(chainID uint64, fromBlock, toBlock uint64) {
	if r.observer == nil {
		return
	}
	r.observer.OnProtocolStarted(ProtocolRunStarted{
		ScannerName: r.scannerName,
		Protocol:    r.protocol,
		ChainID:     chainID,
		StartBlock:  fromBlock,
		EndBlock:    toBlock,
	})
}

func (r *ProtocolRunner) notifyProtocolProgress(chainID uint64, fromBlock, toBlock uint64, stats protocolRunStats) {
	if r.observer == nil {
		return
	}
	r.observer.OnProtocolProgress(ProtocolRunProgress{
		ScannerName:           r.scannerName,
		Protocol:              r.protocol,
		ChainID:               chainID,
		StartBlock:            fromBlock,
		EndBlock:              toBlock,
		CurrentBlock:          stats.currentBlock,
		ObservedTransactions:  stats.observedTransactions,
		CandidateInteractions: stats.candidateInteractions,
		VerifiedInteractions:  stats.verifiedInteractions,
		StrictInteractions:    stats.strictInteractions,
		Findings:              stats.findings,
		VerifiedFindings:      stats.verifiedFindings,
		StrictFindings:        stats.strictFindings,
	})
}

func (r *ProtocolRunner) notifyFinding(chainID uint64, summary scanner.TxSummary) {
	if r.observer == nil {
		return
	}
	protocols := make([]scanner.Protocol, len(summary.Protocols))
	copy(protocols, summary.Protocols)
	r.observer.OnFinding(ProtocolFinding{
		ScannerName:            r.scannerName,
		Protocol:               r.protocol,
		ChainID:                chainID,
		TxHash:                 summary.TxHash,
		BlockNumber:            parseUint64(summary.BlockNumber),
		Candidate:              summary.ContainsCandidateInteraction,
		Verified:               summary.ContainsVerifiedInteraction,
		Strict:                 summary.ContainsVerifiedStrictInteraction,
		InteractionCount:       summary.InteractionCount,
		StrictInteractionCount: summary.StrictInteractionCount,
		ProtocolCount:          summary.ProtocolCount,
		Protocols:              protocols,
	})
}

func (r *ProtocolRunner) notifyProtocolCompleted(chainID uint64, fromBlock, toBlock uint64, stats protocolRunStats) {
	if r.observer == nil {
		return
	}
	r.observer.OnProtocolCompleted(ProtocolRunCompleted{
		ScannerName:           r.scannerName,
		Protocol:              r.protocol,
		ChainID:               chainID,
		StartBlock:            fromBlock,
		EndBlock:              toBlock,
		CurrentBlock:          stats.currentBlock,
		ObservedTransactions:  stats.observedTransactions,
		CandidateInteractions: stats.candidateInteractions,
		VerifiedInteractions:  stats.verifiedInteractions,
		StrictInteractions:    stats.strictInteractions,
		Findings:              stats.findings,
		VerifiedFindings:      stats.verifiedFindings,
		StrictFindings:        stats.strictFindings,
	})
}

func (r *ProtocolRunner) notifyProtocolFailed(chainID uint64, fromBlock, toBlock, currentBlock uint64, err error) {
	if r.observer == nil {
		return
	}
	r.observer.OnProtocolFailed(ProtocolRunFailed{
		ScannerName:  r.scannerName,
		Protocol:     r.protocol,
		ChainID:      chainID,
		StartBlock:   fromBlock,
		EndBlock:     toBlock,
		CurrentBlock: currentBlock,
		Error:        err.Error(),
	})
}

func parseUint64(raw string) uint64 {
	if raw == "" {
		return 0
	}
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok || !value.IsUint64() {
		return 0
	}
	return value.Uint64()
}
