package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http"
	"strconv"
)

func assetData(c *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JSON(200, gin.H{"data": data})
		return
	}
	status, code, msg := 500, "ASSET_FAILED", "资产快照暂时无法读取或保存，请重试。"
	switch {
	case errors.Is(err, service.ErrGoalAssetInput):
		status, code, msg = 400, "GOAL_ASSET_INVALID", "请选择有效的目标、快照与判断日期，说明最多 1000 字；编辑或解除关联需要当前版本。"
	case errors.Is(err, repository.ErrAssetReferenced):
		status, code, msg = 409, "ASSET_REFERENCED", "此快照已关联长期目标，请先在目标详情中解除关联，再删除快照。"
	case errors.Is(err, repository.ErrLifeConflict):
		status, code, msg = 409, "ASSET_REFERENCE_EXISTS", "已有这份快照的财务参考，请编辑现有说明。"
	case errors.Is(err, service.ErrAssetInput):
		status, code, msg = 400, "ASSET_INVALID", "请填写有效日期和至少一项非负金额（最多两位小数、10 位整数），备注最多 1000 字。"
	case errors.Is(err, repository.ErrAssetDateExists):
		status, code, msg = 409, "ASSET_DATE_EXISTS", "这一天已有资产快照，请查看并编辑现有记录。"
	case errors.Is(err, repository.ErrAssetChanged):
		status, code, msg = 409, "ASSET_CHANGED", "记录已被更新，请重新读取后编辑，避免覆盖新内容。"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, code, msg = 404, "ASSET_NOT_FOUND", "资产快照不存在或不属于当前用户。"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": msg, "retryable": status >= 500}})
}
func (h *Handler) AssetList(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	rows, err := h.assets.List(c.Request.Context(), u)
	assetData(c, rows, err)
}
func (h *Handler) AssetGet(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	row, err := h.assets.Get(c.Request.Context(), u, id)
	assetData(c, row, err)
}
func (h *Handler) AssetSave(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	var id uint
	if c.Request.Method == http.MethodPut {
		id, ok = projectPathID(c, "id")
		if !ok {
			return
		}
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	var in service.AssetInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil || decoder.Decode(new(interface{})) != io.EOF {
		assetData(c, nil, service.ErrAssetInput)
		return
	}
	row, err := h.assets.Save(c.Request.Context(), u, id, in)
	assetData(c, row, err)
}
func (h *Handler) AssetDelete(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	version, err := strconv.ParseUint(c.Query("version"), 10, 32)
	if err != nil {
		assetData(c, nil, service.ErrAssetInput)
		return
	}
	err = h.assets.Delete(c.Request.Context(), u, id, uint(version))
	assetData(c, gin.H{"deleted": err == nil}, err)
}
