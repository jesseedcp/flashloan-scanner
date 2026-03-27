package service

import (
	"time"

	scannerorchestrator "github.com/cpchain-network/flashloan-scanner/scanner/orchestrator"
)

type JobObserver struct {
	jobID   string
	manager *JobManager
}

func NewJobObserver(jobID string, manager *JobManager) *JobObserver {
	return &JobObserver{
		jobID:   jobID,
		manager: manager,
	}
}

func (o *JobObserver) OnProtocolStarted(event scannerorchestrator.ProtocolRunStarted) {
	if o == nil || o.manager == nil {
		return
	}
	_ = o.manager.RecordProtocolStarted(o.jobID, event.Protocol, event.StartBlock, event.EndBlock)
}

func (o *JobObserver) OnProtocolProgress(event scannerorchestrator.ProtocolRunProgress) {
	if o == nil || o.manager == nil {
		return
	}
	_ = o.manager.RecordProtocolProgress(o.jobID, ProtocolProgressInput{
		Protocol:        event.Protocol,
		CurrentBlock:    event.CurrentBlock,
		FoundCandidates: event.Findings,
		FoundVerified:   event.VerifiedFindings,
		FoundStrict:     event.StrictFindings,
	})
}

func (o *JobObserver) OnFinding(event scannerorchestrator.ProtocolFinding) {
	if o == nil || o.manager == nil {
		return
	}
	_ = o.manager.RecordFinding(o.jobID, JobFinding{
		TxHash:                 event.TxHash,
		Protocol:               event.Protocol,
		BlockNumber:            event.BlockNumber,
		Candidate:              event.Candidate,
		Verified:               event.Verified,
		Strict:                 event.Strict,
		InteractionCount:       event.InteractionCount,
		StrictInteractionCount: event.StrictInteractionCount,
		ProtocolCount:          event.ProtocolCount,
		Protocols:              event.Protocols,
		CreatedAt:              time.Now().UTC(),
	})
}

func (o *JobObserver) OnProtocolCompleted(event scannerorchestrator.ProtocolRunCompleted) {
	if o == nil || o.manager == nil {
		return
	}
	_ = o.manager.RecordProtocolCompleted(o.jobID, ProtocolProgressInput{
		Protocol:        event.Protocol,
		CurrentBlock:    event.CurrentBlock,
		FoundCandidates: event.Findings,
		FoundVerified:   event.VerifiedFindings,
		FoundStrict:     event.StrictFindings,
	})
}

func (o *JobObserver) OnProtocolFailed(event scannerorchestrator.ProtocolRunFailed) {
	if o == nil || o.manager == nil {
		return
	}
	_ = o.manager.RecordProtocolFailed(o.jobID, event.Protocol, event.CurrentBlock, event.Error)
}
