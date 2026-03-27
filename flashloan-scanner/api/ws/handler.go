package ws

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	apiservice "github.com/cpchain-network/flashloan-scanner/api/service"
)

type Handler struct {
	jobManager   *apiservice.JobManager
	runnerBridge *apiservice.RunnerBridge
}

func NewHandler(jobManager *apiservice.JobManager, runnerBridge *apiservice.RunnerBridge) *Handler {
	return &Handler{
		jobManager:   jobManager,
		runnerBridge: runnerBridge,
	}
}

func (h *Handler) HandleScan(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	writer := &safeConnWriter{conn: conn}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	var (
		unsubscribe   func()
		forwardCancel context.CancelFunc
	)
	defer func() {
		if forwardCancel != nil {
			forwardCancel()
		}
		if unsubscribe != nil {
			unsubscribe()
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		message, err := DecodeIncomingMessage(data)
		if err != nil {
			_ = writer.WriteJSON(ErrorMessage(err))
			continue
		}

		switch message.Type {
		case MessageTypeStartScan:
			params, err := DecodeStartScanPayload(message.Payload)
			if err != nil {
				_ = writer.WriteJSON(ErrorMessage(err))
				continue
			}

			job, err := h.jobManager.CreateJob(params)
			if err != nil {
				_ = writer.WriteJSON(ErrorMessage(err))
				continue
			}

			events, unsubscribeFn, err := h.jobManager.Subscribe(job.JobID, 128)
			if err != nil {
				_ = writer.WriteJSON(ErrorMessage(err))
				continue
			}

			if forwardCancel != nil {
				forwardCancel()
			}
			if unsubscribe != nil {
				unsubscribe()
			}

			subCtx, subCancel := context.WithCancel(ctx)
			forwardCancel = subCancel
			unsubscribe = unsubscribeFn
			go forwardEvents(subCtx, writer, events)

			if err := h.runnerBridge.RunJobAsync(ctx, h.jobManager, job.JobID, params); err != nil {
				subCancel()
				unsubscribeFn()
				forwardCancel = nil
				unsubscribe = nil
				_ = writer.WriteJSON(ErrorMessage(err))
				continue
			}
		default:
			_ = writer.WriteJSON(ErrorMessage(&unsupportedMessageError{messageType: message.Type}))
		}
	}
}

func forwardEvents(ctx context.Context, writer *safeConnWriter, events <-chan apiservice.JobEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writer.WriteJSON(FromJobEvent(event)); err != nil {
				return
			}
		}
	}
}

type safeConnWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *safeConnWriter) WriteJSON(payload any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(payload)
}

type unsupportedMessageError struct {
	messageType string
}

func (e *unsupportedMessageError) Error() string {
	return "unsupported message type: " + e.messageType
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}
