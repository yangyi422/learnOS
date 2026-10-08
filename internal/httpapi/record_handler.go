package httpapi

import (
	"net/http"

	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListRecords(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := optionalProjectID(c)
	if !ok {
		return
	}
	status := c.DefaultQuery("status", "active")
	if status != "active" && status != "archived" {
		inboxError(c, service.ErrInvalidRecordInput)
		return
	}
	records, err := h.records.List(c.Request.Context(), userID, projectID, status == "archived")
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": records})
}
func (h *Handler) GetRecord(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	record, err := h.records.Find(c.Request.Context(), userID, id)
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": record})
}
func (h *Handler) CreateRecord(c *gin.Context) { h.saveRecord(c, false) }
func (h *Handler) UpdateRecord(c *gin.Context) { h.saveRecord(c, true) }
func (h *Handler) saveRecord(c *gin.Context, updating bool) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	var id uint
	if updating {
		id, ok = projectPathID(c, "id")
		if !ok {
			return
		}
	}
	var input service.RecordInput
	if c.ShouldBindJSON(&input) != nil {
		inboxError(c, service.ErrInvalidRecordInput)
		return
	}
	record, err := h.records.Save(c.Request.Context(), userID, id, input)
	if err != nil {
		inboxError(c, err)
		return
	}
	status := http.StatusCreated
	if updating {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"data": record})
}
func (h *Handler) ConvertInboxItemToRecord(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input service.RecordInput
	if c.ShouldBindJSON(&input) != nil {
		inboxError(c, service.ErrInvalidRecordInput)
		return
	}
	item, record, err := h.inbox.ConvertToRecord(c.Request.Context(), userID, id, input)
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"item": item, "record": record}})
}
func (h *Handler) ArchiveRecord(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input struct {
		Archived *bool `json:"archived"`
	}
	if c.ShouldBindJSON(&input) != nil || input.Archived == nil {
		inboxError(c, service.ErrInvalidRecordInput)
		return
	}
	if err := h.records.Archive(c.Request.Context(), userID, id, *input.Archived); err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"archived": *input.Archived}})
}
func (h *Handler) DeleteRecord(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.records.Delete(c.Request.Context(), userID, id); err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}
