package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Transactions(c *gin.Context) {
	chainID, err := h.parseChainID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	limit := 50
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err))
			return
		}
		limit = parsed
	}

	strictOnly := c.Query("strict_only") == "true"
	items, err := h.queryService.ListTransactions(chainID, strictOnly, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	writeOK(c, items)
}

func (h *Handler) TransactionDetail(c *gin.Context) {
	chainID, err := h.parseChainID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	detail, err := h.queryService.GetTransactionDetail(c.Request.Context(), chainID, c.Param("txHash"))
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if detail == nil {
		writeError(c, http.StatusNotFound, fmt.Errorf("transaction not found"))
		return
	}
	writeOK(c, detail)
}
