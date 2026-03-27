package ws

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiservice "github.com/cpchain-network/flashloan-scanner/api/service"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

const (
	MessageTypeStartScan = "start_scan"
	MessageTypeError     = "error"
)

type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type OutgoingMessage struct {
	Type      string    `json:"type"`
	JobID     string    `json:"job_id,omitempty"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type StartScanPayload struct {
	ChainID      uint64   `json:"chain_id"`
	StartBlock   uint64   `json:"start_block"`
	EndBlock     uint64   `json:"end_block"`
	TraceEnabled bool     `json:"trace_enabled"`
	Protocols    []string `json:"protocols"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

func DecodeIncomingMessage(data []byte) (IncomingMessage, error) {
	var message IncomingMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return IncomingMessage{}, err
	}
	if message.Type == "" {
		return IncomingMessage{}, fmt.Errorf("message type is required")
	}
	return message, nil
}

func DecodeStartScanPayload(raw json.RawMessage) (apiservice.ScanJobParams, error) {
	var payload StartScanPayload
	if len(raw) == 0 {
		return apiservice.ScanJobParams{}, fmt.Errorf("start_scan payload is required")
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return apiservice.ScanJobParams{}, err
	}
	if payload.ChainID == 0 {
		return apiservice.ScanJobParams{}, fmt.Errorf("chain_id is required")
	}
	if payload.EndBlock < payload.StartBlock {
		return apiservice.ScanJobParams{}, fmt.Errorf("end_block (%d) is less than start_block (%d)", payload.EndBlock, payload.StartBlock)
	}

	protocols := payload.Protocols
	if len(protocols) == 0 {
		protocols = []string{
			string(scanner.ProtocolAaveV3),
			string(scanner.ProtocolBalancerV2),
			string(scanner.ProtocolUniswapV2),
		}
	}

	normalizedProtocols := make([]scanner.Protocol, 0, len(protocols))
	for _, protocol := range protocols {
		switch scanner.Protocol(strings.ToLower(strings.TrimSpace(protocol))) {
		case scanner.ProtocolAaveV3:
			normalizedProtocols = append(normalizedProtocols, scanner.ProtocolAaveV3)
		case scanner.ProtocolBalancerV2:
			normalizedProtocols = append(normalizedProtocols, scanner.ProtocolBalancerV2)
		case scanner.ProtocolUniswapV2:
			normalizedProtocols = append(normalizedProtocols, scanner.ProtocolUniswapV2)
		default:
			return apiservice.ScanJobParams{}, fmt.Errorf("unsupported protocol %s", protocol)
		}
	}

	return apiservice.ScanJobParams{
		ChainID:      payload.ChainID,
		StartBlock:   payload.StartBlock,
		EndBlock:     payload.EndBlock,
		TraceEnabled: payload.TraceEnabled,
		Protocols:    normalizedProtocols,
	}, nil
}

func FromJobEvent(event apiservice.JobEvent) OutgoingMessage {
	return OutgoingMessage{
		Type:      event.Type,
		JobID:     event.JobID,
		Payload:   event.Payload,
		Timestamp: event.Timestamp,
	}
}

func ErrorMessage(err error) OutgoingMessage {
	return OutgoingMessage{
		Type: MessageTypeError,
		Payload: ErrorPayload{
			Message: err.Error(),
		},
		Timestamp: time.Now().UTC(),
	}
}
