package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func TestJobManagerLifecycle(t *testing.T) {
	manager := NewJobManager()

	job, err := manager.CreateJob(ScanJobParams{
		ChainID:      1,
		StartBlock:   100,
		EndBlock:     110,
		TraceEnabled: false,
		Protocols:    []scanner.Protocol{scanner.ProtocolAaveV3},
	})
	require.NoError(t, err)
	require.Equal(t, JobStatusPending, job.Status)

	events, unsubscribe, err := manager.Subscribe(job.JobID, 16)
	require.NoError(t, err)
	defer unsubscribe()

	require.NoError(t, manager.MarkJobStarted(job.JobID))
	require.NoError(t, manager.RecordProtocolStarted(job.JobID, scanner.ProtocolAaveV3, 100, 110))
	require.NoError(t, manager.RecordFinding(job.JobID, JobFinding{
		TxHash:      "0xabc",
		Protocol:    scanner.ProtocolAaveV3,
		BlockNumber: 105,
		Candidate:   true,
		Verified:    true,
		Strict:      false,
		CreatedAt:   time.Now().UTC(),
	}))
	require.NoError(t, manager.RecordProtocolProgress(job.JobID, ProtocolProgressInput{
		Protocol:        scanner.ProtocolAaveV3,
		CurrentBlock:    105,
		FoundCandidates: 1,
		FoundVerified:   1,
		FoundStrict:     0,
	}))
	require.NoError(t, manager.RecordProtocolCompleted(job.JobID, ProtocolProgressInput{
		Protocol:        scanner.ProtocolAaveV3,
		CurrentBlock:    110,
		FoundCandidates: 1,
		FoundVerified:   1,
		FoundStrict:     0,
	}))
	require.NoError(t, manager.MarkJobCompleted(job.JobID))

	overview, ok := manager.GetJob(job.JobID)
	require.True(t, ok)
	require.Equal(t, JobStatusCompleted, overview.Status)
	require.Equal(t, 1, overview.TotalCandidates)
	require.Equal(t, 1, overview.TotalVerified)
	require.Equal(t, 0, overview.TotalStrict)
	require.Equal(t, 1, overview.CompletedProtocols)
	require.Equal(t, JobStatusCompleted, overview.Protocols[string(scanner.ProtocolAaveV3)].Status)

	findings, ok := manager.GetJobFindings(job.JobID)
	require.True(t, ok)
	require.Len(t, findings, 1)
	require.Equal(t, "0xabc", findings[0].TxHash)

	jobState := manager.jobs[job.JobID]
	require.NotNil(t, jobState)
	foundFindingLog := false
	for _, entry := range jobState.Logs {
		if strings.Contains(entry.Message, "found tx 0xabc") {
			require.Equal(t, uint64(105), entry.BlockNumber)
			foundFindingLog = true
			break
		}
	}
	require.True(t, foundFindingLog)

	collected := collectEvents(events)
	require.Contains(t, collected, EventTypeJobStarted)
	require.Contains(t, collected, EventTypeProtocolProgress)
	require.Contains(t, collected, EventTypeFinding)
	require.Contains(t, collected, EventTypeProtocolCompleted)
	require.Contains(t, collected, EventTypeJobCompleted)
}

func TestJobManagerRejectsConcurrentActiveJob(t *testing.T) {
	manager := NewJobManager()

	job, err := manager.CreateJob(ScanJobParams{
		ChainID:    1,
		StartBlock: 100,
		EndBlock:   110,
		Protocols:  []scanner.Protocol{scanner.ProtocolAaveV3},
	})
	require.NoError(t, err)
	require.NoError(t, manager.MarkJobStarted(job.JobID))

	_, err = manager.CreateJob(ScanJobParams{
		ChainID:    1,
		StartBlock: 200,
		EndBlock:   210,
		Protocols:  []scanner.Protocol{scanner.ProtocolBalancerV2},
	})
	require.EqualError(t, err, "a scan job is already active")
}

func collectEvents(events <-chan JobEvent) []string {
	out := make([]string, 0, 16)
	for {
		select {
		case event := <-events:
			out = append(out, event.Type)
		default:
			return out
		}
	}
	return out
}
