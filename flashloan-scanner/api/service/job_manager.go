package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

const (
	defaultSubscriberBuffer = 64
	maxJobLogs              = 200
)

type JobManager struct {
	mu          sync.RWMutex
	jobs        map[string]*ScanJobState
	subscribers map[string]map[chan JobEvent]struct{}
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs:        make(map[string]*ScanJobState),
		subscribers: make(map[string]map[chan JobEvent]struct{}),
	}
}

func (m *JobManager) CreateJob(params ScanJobParams) (JobOverview, error) {
	if params.ChainID == 0 {
		return JobOverview{}, fmt.Errorf("chain id is required")
	}
	if params.EndBlock < params.StartBlock {
		return JobOverview{}, fmt.Errorf("end block (%d) is less than start block (%d)", params.EndBlock, params.StartBlock)
	}
	protocols, err := normalizeProtocols(params.Protocols)
	if err != nil {
		return JobOverview{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range m.jobs {
		if job.Status == JobStatusPending || job.Status == JobStatusRunning {
			return JobOverview{}, fmt.Errorf("a scan job is already active")
		}
	}

	job := &ScanJobState{
		JobID:           uuid.NewString(),
		Status:          JobStatusPending,
		ChainID:         params.ChainID,
		StartBlock:      params.StartBlock,
		EndBlock:        params.EndBlock,
		TraceEnabled:    params.TraceEnabled,
		Protocols:       make(map[scanner.Protocol]*ProtocolJobState, len(protocols)),
		Findings:        make([]JobFinding, 0, 32),
		Logs:            make([]JobLogEntry, 0, 32),
		TotalCandidates: 0,
		TotalVerified:   0,
		TotalStrict:     0,
	}
	for _, protocol := range protocols {
		job.Protocols[protocol] = &ProtocolJobState{
			Protocol:   protocol,
			Status:     JobStatusPending,
			StartBlock: params.StartBlock,
			EndBlock:   params.EndBlock,
		}
	}

	m.jobs[job.JobID] = job
	return cloneJobOverview(job), nil
}

func (m *JobManager) GetJob(jobID string) (JobOverview, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return JobOverview{}, false
	}
	return cloneJobOverview(job), true
}

func (m *JobManager) GetJobFindings(jobID string) ([]JobFinding, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, false
	}
	return cloneFindings(job.Findings), true
}

func (m *JobManager) Subscribe(jobID string, buffer int) (<-chan JobEvent, func(), error) {
	if buffer <= 0 {
		buffer = defaultSubscriberBuffer
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.jobs[jobID]; !ok {
		return nil, nil, fmt.Errorf("job %s not found", jobID)
	}

	ch := make(chan JobEvent, buffer)
	if m.subscribers[jobID] == nil {
		m.subscribers[jobID] = make(map[chan JobEvent]struct{})
	}
	m.subscribers[jobID][ch] = struct{}{}

	unsubscribe := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs := m.subscribers[jobID]
		if subs == nil {
			return
		}
		if _, ok := subs[ch]; !ok {
			return
		}
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(m.subscribers, jobID)
		}
	}

	return ch, unsubscribe, nil
}

func (m *JobManager) MarkJobStarted(jobID string) error {
	m.mu.Lock()
	job, err := m.getJobLocked(jobID)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	job.Status = JobStatusRunning
	job.StartedAt = now
	logEntry := appendLogLocked(job, "info", "", 0, "scan job started")
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeJobStarted, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) MarkJobCompleted(jobID string) error {
	m.mu.Lock()
	job, err := m.getJobLocked(jobID)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	job.Status = JobStatusCompleted
	job.FinishedAt = &now
	logEntry := appendLogLocked(job, "info", "", 0, "scan job completed")
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeJobCompleted, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) MarkJobFailed(jobID string, message string) error {
	m.mu.Lock()
	job, err := m.getJobLocked(jobID)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	job.Status = JobStatusFailed
	job.Error = message
	job.FinishedAt = &now
	logEntry := appendLogLocked(job, "error", "", 0, fmt.Sprintf("scan job failed: %s", message))
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeJobFailed, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) RecordProtocolStarted(jobID string, protocol scanner.Protocol, startBlock, endBlock uint64) error {
	m.mu.Lock()
	job, protocolState, err := m.getProtocolLocked(jobID, protocol)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	protocolState.Status = JobStatusRunning
	protocolState.StartBlock = startBlock
	protocolState.EndBlock = endBlock
	protocolState.CurrentBlock = startBlock
	protocolState.StartedAt = &now
	logEntry := appendLogLocked(job, "info", protocol, startBlock, fmt.Sprintf("%s scan started", protocol))
	protocolSnapshot := cloneProtocolSnapshot(protocolState)
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeProtocolProgress, protocolSnapshot)
	m.broadcast(jobID, EventTypeJobProgress, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) RecordProtocolProgress(jobID string, event ProtocolProgressInput) error {
	m.mu.Lock()
	job, protocolState, err := m.getProtocolLocked(jobID, event.Protocol)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	protocolState.Status = JobStatusRunning
	protocolState.CurrentBlock = event.CurrentBlock
	protocolState.FoundCandidates = event.FoundCandidates
	protocolState.FoundVerified = event.FoundVerified
	protocolState.FoundStrict = event.FoundStrict
	recalculateJobTotalsLocked(job)
	protocolSnapshot := cloneProtocolSnapshot(protocolState)
	overview := cloneJobOverview(job)
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeProtocolProgress, protocolSnapshot)
	m.broadcast(jobID, EventTypeJobProgress, overview)
	return nil
}

func (m *JobManager) RecordFinding(jobID string, finding JobFinding) error {
	m.mu.Lock()
	job, protocolState, err := m.getProtocolLocked(jobID, finding.Protocol)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	job.Findings = append([]JobFinding{cloneFinding(finding)}, job.Findings...)
	clonedFinding := cloneFinding(finding)
	protocolState.LatestFinding = &clonedFinding
	logEntry := appendLogLocked(job, "info", finding.Protocol, finding.BlockNumber, fmt.Sprintf("%s found tx %s", finding.Protocol, finding.TxHash))
	findingSnapshot := cloneFinding(finding)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeFinding, findingSnapshot)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) RecordProtocolCompleted(jobID string, event ProtocolProgressInput) error {
	m.mu.Lock()
	job, protocolState, err := m.getProtocolLocked(jobID, event.Protocol)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	protocolState.Status = JobStatusCompleted
	protocolState.CurrentBlock = event.CurrentBlock
	protocolState.FoundCandidates = event.FoundCandidates
	protocolState.FoundVerified = event.FoundVerified
	protocolState.FoundStrict = event.FoundStrict
	protocolState.FinishedAt = &now
	recalculateJobTotalsLocked(job)
	logEntry := appendLogLocked(job, "info", event.Protocol, event.CurrentBlock, fmt.Sprintf("%s scan completed", event.Protocol))
	protocolSnapshot := cloneProtocolSnapshot(protocolState)
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeProtocolCompleted, protocolSnapshot)
	m.broadcast(jobID, EventTypeJobProgress, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

func (m *JobManager) RecordProtocolFailed(jobID string, protocol scanner.Protocol, currentBlock uint64, message string) error {
	m.mu.Lock()
	job, protocolState, err := m.getProtocolLocked(jobID, protocol)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	now := time.Now().UTC()
	protocolState.Status = JobStatusFailed
	protocolState.CurrentBlock = currentBlock
	protocolState.Error = message
	protocolState.FinishedAt = &now
	recalculateJobTotalsLocked(job)
	logEntry := appendLogLocked(job, "error", protocol, currentBlock, fmt.Sprintf("%s scan failed: %s", protocol, message))
	protocolSnapshot := cloneProtocolSnapshot(protocolState)
	overview := cloneJobOverview(job)
	logSnapshot := logEntry
	m.mu.Unlock()

	m.broadcast(jobID, EventTypeProtocolFailed, protocolSnapshot)
	m.broadcast(jobID, EventTypeJobProgress, overview)
	m.broadcast(jobID, EventTypeLog, logSnapshot)
	return nil
}

type ProtocolProgressInput struct {
	Protocol        scanner.Protocol
	CurrentBlock    uint64
	FoundCandidates int
	FoundVerified   int
	FoundStrict     int
}

func (m *JobManager) getJobLocked(jobID string) (*ScanJobState, error) {
	job, ok := m.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

func (m *JobManager) getProtocolLocked(jobID string, protocol scanner.Protocol) (*ScanJobState, *ProtocolJobState, error) {
	job, err := m.getJobLocked(jobID)
	if err != nil {
		return nil, nil, err
	}
	protocolState, ok := job.Protocols[protocol]
	if !ok {
		return nil, nil, fmt.Errorf("protocol %s not configured for job %s", protocol, jobID)
	}
	return job, protocolState, nil
}

func (m *JobManager) broadcast(jobID, eventType string, payload any) {
	event := JobEvent{
		Type:      eventType,
		JobID:     jobID,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	m.mu.RLock()
	subscribers := m.subscribers[jobID]
	channels := make([]chan JobEvent, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

func appendLogLocked(job *ScanJobState, level string, protocol scanner.Protocol, blockNumber uint64, message string) JobLogEntry {
	entry := JobLogEntry{
		Level:       level,
		Message:     message,
		Protocol:    protocol,
		BlockNumber: blockNumber,
		Timestamp:   time.Now().UTC(),
	}
	job.Logs = append([]JobLogEntry{entry}, job.Logs...)
	if len(job.Logs) > maxJobLogs {
		job.Logs = job.Logs[:maxJobLogs]
	}
	return entry
}

func recalculateJobTotalsLocked(job *ScanJobState) {
	job.TotalCandidates = 0
	job.TotalVerified = 0
	job.TotalStrict = 0
	for _, protocolState := range job.Protocols {
		job.TotalCandidates += protocolState.FoundCandidates
		job.TotalVerified += protocolState.FoundVerified
		job.TotalStrict += protocolState.FoundStrict
	}
}

func cloneJobOverview(job *ScanJobState) JobOverview {
	overview := JobOverview{
		JobID:              job.JobID,
		Status:             job.Status,
		ChainID:            job.ChainID,
		StartBlock:         job.StartBlock,
		EndBlock:           job.EndBlock,
		TraceEnabled:       job.TraceEnabled,
		TotalCandidates:    job.TotalCandidates,
		TotalVerified:      job.TotalVerified,
		TotalStrict:        job.TotalStrict,
		CompletedProtocols: completedProtocols(job.Protocols),
		Protocols:          make(map[string]ProtocolJobSnapshot, len(job.Protocols)),
		StartedAt:          job.StartedAt,
		FinishedAt:         cloneTimePtr(job.FinishedAt),
		Error:              job.Error,
	}
	for protocol, protocolState := range job.Protocols {
		overview.Protocols[string(protocol)] = cloneProtocolSnapshot(protocolState)
	}
	return overview
}

func cloneProtocolSnapshot(state *ProtocolJobState) ProtocolJobSnapshot {
	snapshot := ProtocolJobSnapshot{
		Protocol:        state.Protocol,
		Status:          state.Status,
		StartBlock:      state.StartBlock,
		EndBlock:        state.EndBlock,
		CurrentBlock:    state.CurrentBlock,
		FoundCandidates: state.FoundCandidates,
		FoundVerified:   state.FoundVerified,
		FoundStrict:     state.FoundStrict,
		Error:           state.Error,
		StartedAt:       cloneTimePtr(state.StartedAt),
		FinishedAt:      cloneTimePtr(state.FinishedAt),
	}
	if state.LatestFinding != nil {
		finding := cloneFinding(*state.LatestFinding)
		snapshot.LatestFinding = &finding
	}
	return snapshot
}

func cloneFindings(items []JobFinding) []JobFinding {
	out := make([]JobFinding, 0, len(items))
	for _, item := range items {
		out = append(out, cloneFinding(item))
	}
	return out
}

func cloneFinding(item JobFinding) JobFinding {
	protocols := make([]scanner.Protocol, len(item.Protocols))
	copy(protocols, item.Protocols)
	item.Protocols = protocols
	return item
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func completedProtocols(protocols map[scanner.Protocol]*ProtocolJobState) int {
	count := 0
	for _, protocolState := range protocols {
		if protocolState.Status == JobStatusCompleted {
			count++
		}
	}
	return count
}

func normalizeProtocols(protocols []scanner.Protocol) ([]scanner.Protocol, error) {
	if len(protocols) == 0 {
		return nil, fmt.Errorf("at least one protocol is required")
	}

	seen := make(map[scanner.Protocol]struct{}, len(protocols))
	out := make([]scanner.Protocol, 0, len(protocols))
	for _, protocol := range protocols {
		switch protocol {
		case scanner.ProtocolAaveV3, scanner.ProtocolBalancerV2, scanner.ProtocolUniswapV2:
		default:
			return nil, fmt.Errorf("unsupported protocol %s", protocol)
		}
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		out = append(out, protocol)
	}
	return out, nil
}
