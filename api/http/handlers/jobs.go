package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) JobResults(c *gin.Context) {
	results, err := h.queryService.GetJobResults(c.Param("jobId"))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	writeOK(c, results)
}
