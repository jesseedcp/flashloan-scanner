package orchestrator

import "github.com/cpchain-network/flashloan-scanner/scanner"

type ProtocolRunObserver interface {
	OnProtocolStarted(event ProtocolRunStarted)
	OnProtocolProgress(event ProtocolRunProgress)
	OnFinding(event ProtocolFinding)
	OnProtocolCompleted(event ProtocolRunCompleted)
	OnProtocolFailed(event ProtocolRunFailed)
}

type ProtocolRunStarted struct {
	ScannerName string
	Protocol    scanner.Protocol
	ChainID     uint64
	StartBlock  uint64
	EndBlock    uint64
}

type ProtocolRunProgress struct {
	ScannerName           string
	Protocol              scanner.Protocol
	ChainID               uint64
	StartBlock            uint64
	EndBlock              uint64
	CurrentBlock          uint64
	ObservedTransactions  int
	CandidateInteractions int
	VerifiedInteractions  int
	StrictInteractions    int
	Findings              int
	VerifiedFindings      int
	StrictFindings        int
}

type ProtocolFinding struct {
	ScannerName            string
	Protocol               scanner.Protocol
	ChainID                uint64
	TxHash                 string
	BlockNumber            uint64
	Candidate              bool
	Verified               bool
	Strict                 bool
	InteractionCount       int
	StrictInteractionCount int
	ProtocolCount          int
	Protocols              []scanner.Protocol
}

type ProtocolRunCompleted struct {
	ScannerName           string
	Protocol              scanner.Protocol
	ChainID               uint64
	StartBlock            uint64
	EndBlock              uint64
	CurrentBlock          uint64
	ObservedTransactions  int
	CandidateInteractions int
	VerifiedInteractions  int
	StrictInteractions    int
	Findings              int
	VerifiedFindings      int
	StrictFindings        int
}

type ProtocolRunFailed struct {
	ScannerName  string
	Protocol     scanner.Protocol
	ChainID      uint64
	StartBlock   uint64
	EndBlock     uint64
	CurrentBlock uint64
	Error        string
}
