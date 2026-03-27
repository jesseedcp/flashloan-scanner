package service

import (
	"time"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

const (
	EventTypeJobStarted        = "job_started"
	EventTypeJobProgress       = "job_progress"
	EventTypeFinding           = "finding"
	EventTypeProtocolProgress  = "protocol_progress"
	EventTypeProtocolCompleted = "protocol_completed"
	EventTypeProtocolFailed    = "protocol_failed"
	EventTypeJobCompleted      = "job_completed"
	EventTypeJobFailed         = "job_failed"
	EventTypeLog               = "log"
)

type ScanJobParams struct {
	ChainID      uint64
	StartBlock   uint64
	EndBlock     uint64
	TraceEnabled bool
	Protocols    []scanner.Protocol
}

type JobEvent struct {
	Type      string    `json:"type"`
	JobID     string    `json:"job_id"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type ScanJobState struct {
	JobID           string
	Status          JobStatus
	ChainID         uint64
	StartBlock      uint64
	EndBlock        uint64
	TraceEnabled    bool
	Protocols       map[scanner.Protocol]*ProtocolJobState
	Findings        []JobFinding
	Logs            []JobLogEntry
	TotalCandidates int
	TotalVerified   int
	TotalStrict     int
	StartedAt       time.Time
	FinishedAt      *time.Time
	Error           string
}

type ProtocolJobState struct {
	Protocol        scanner.Protocol
	Status          JobStatus
	StartBlock      uint64
	EndBlock        uint64
	CurrentBlock    uint64
	FoundCandidates int
	FoundVerified   int
	FoundStrict     int
	LatestFinding   *JobFinding
	Error           string
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

type JobFinding struct {
	TxHash                 string             `json:"tx_hash"`
	Protocol               scanner.Protocol   `json:"protocol"`
	BlockNumber            uint64             `json:"block_number"`
	Candidate              bool               `json:"candidate"`
	Verified               bool               `json:"verified"`
	Strict                 bool               `json:"strict"`
	InteractionCount       int                `json:"interaction_count"`
	StrictInteractionCount int                `json:"strict_interaction_count"`
	ProtocolCount          int                `json:"protocol_count"`
	Protocols              []scanner.Protocol `json:"protocols"`
	CreatedAt              time.Time          `json:"created_at"`
}

type JobLogEntry struct {
	Level       string           `json:"level"`
	Message     string           `json:"message"`
	Protocol    scanner.Protocol `json:"protocol,omitempty"`
	BlockNumber uint64           `json:"block_number,omitempty"`
	Timestamp   time.Time        `json:"timestamp"`
}

type JobOverview struct {
	JobID              string                         `json:"job_id"`
	Status             JobStatus                      `json:"status"`
	ChainID            uint64                         `json:"chain_id"`
	StartBlock         uint64                         `json:"start_block"`
	EndBlock           uint64                         `json:"end_block"`
	TraceEnabled       bool                           `json:"trace_enabled"`
	TotalCandidates    int                            `json:"total_candidates"`
	TotalVerified      int                            `json:"total_verified"`
	TotalStrict        int                            `json:"total_strict"`
	CompletedProtocols int                            `json:"completed_protocols"`
	Protocols          map[string]ProtocolJobSnapshot `json:"protocols"`
	StartedAt          time.Time                      `json:"started_at"`
	FinishedAt         *time.Time                     `json:"finished_at,omitempty"`
	Error              string                         `json:"error,omitempty"`
}

type ProtocolJobSnapshot struct {
	Protocol        scanner.Protocol `json:"protocol"`
	Status          JobStatus        `json:"status"`
	StartBlock      uint64           `json:"start_block"`
	EndBlock        uint64           `json:"end_block"`
	CurrentBlock    uint64           `json:"current_block"`
	FoundCandidates int              `json:"found_candidates"`
	FoundVerified   int              `json:"found_verified"`
	FoundStrict     int              `json:"found_strict"`
	LatestFinding   *JobFinding      `json:"latest_finding,omitempty"`
	Error           string           `json:"error,omitempty"`
	StartedAt       *time.Time       `json:"started_at,omitempty"`
	FinishedAt      *time.Time       `json:"finished_at,omitempty"`
}
