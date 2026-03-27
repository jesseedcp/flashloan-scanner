package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Summary(c *gin.Context) {
	chainID, err := h.parseChainID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	summary, err := h.queryService.GetSummary(chainID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	writeOK(c, summary)
}
