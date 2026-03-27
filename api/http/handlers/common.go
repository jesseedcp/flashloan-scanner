package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	apiservice "github.com/cpchain-network/flashloan-scanner/api/service"
)

type Handler struct {
	queryService   *apiservice.QueryService
	defaultChainID uint64
}

func New(queryService *apiservice.QueryService, defaultChainID uint64) *Handler {
	return &Handler{
		queryService:   queryService,
		defaultChainID: defaultChainID,
	}
}

func (h *Handler) parseChainID(c *gin.Context) (uint64, error) {
	raw := c.Query("chain_id")
	if raw == "" {
		return h.defaultChainID, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func writeOK(c *gin.Context, payload any) {
	c.JSON(http.StatusOK, payload)
}
