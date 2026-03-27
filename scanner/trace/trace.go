package trace

import "context"

type CallFrame struct {
	Type         string      `json:"type"`
	From         string      `json:"from"`
	To           string      `json:"to"`
	Input        string      `json:"input"`
	Output       string      `json:"output"`
	Error        string      `json:"error"`
	RevertReason string      `json:"revertReason"`
	Calls        []CallFrame `json:"calls"`
}

type Provider interface {
	TraceTransaction(ctx context.Context, txHash string) (*CallFrame, error)
}
