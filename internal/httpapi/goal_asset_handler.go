package httpapi

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"learnos/internal/service"
	"net/http"
	"strconv"
)

func (h *Handler) GoalAssets(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	goal, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	rows, err := h.assets.GoalReferences(c.Request.Context(), user, goal)
	assetData(c, rows, err)
}
func (h *Handler) AssetGoals(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	asset, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	rows, err := h.assets.ReferencingGoals(c.Request.Context(), user, asset)
	assetData(c, rows, err)
}
func (h *Handler) SaveGoalAsset(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	goal, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	asset, ok := projectPathID(c, "assetId")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	var in service.GoalAssetInput
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(new(interface{})) != io.EOF {
		assetData(c, nil, service.ErrGoalAssetInput)
		return
	}
	row, err := h.assets.SaveReference(c.Request.Context(), user, goal, asset, in, c.Request.Method == http.MethodPost)
	assetData(c, row, err)
}
func (h *Handler) DeleteGoalAsset(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	goal, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	asset, ok := projectPathID(c, "assetId")
	if !ok {
		return
	}
	version, err := strconv.ParseUint(c.Query("version"), 10, 32)
	if err != nil {
		assetData(c, nil, service.ErrGoalAssetInput)
		return
	}
	err = h.assets.DeleteReference(c.Request.Context(), user, goal, asset, uint(version))
	assetData(c, gin.H{"deleted": err == nil}, err)
}
