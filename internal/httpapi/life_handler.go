package httpapi

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http"
)

func lifeError(c *gin.Context, err error) {
	status, code, message, retry := 500, "LIFE_SAVE_FAILED", "生活档案暂时无法保存，请重试。", true
	if errors.Is(err, service.ErrInvalidLifeInput) {
		status, code, message, retry = 400, "INVALID_LIFE_INPUT", "请检查名称、日期、领域和链接。", false
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		status, code, message, retry = 404, "LIFE_NOT_FOUND", "记录或关联对象已不存在，或不属于当前用户。", false
	} else if errors.Is(err, repository.ErrLifeSource) {
		status, code, message, retry = 409, "LIFE_SOURCE_INELIGIBLE", "来源尚未正式完成，不能作为成果收录。", false
	} else if errors.Is(err, repository.ErrLifeConflict) {
		status, code, message, retry = 409, "LIFE_CONFLICT", "内容已经整理或请求与已保存内容不一致，请刷新查看。", false
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "retryable": retry}})
}
func (h *Handler) ListLifeEvents(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	m := c.Query("milestones")
	if m != "" && m != "true" && m != "false" {
		lifeError(c, service.ErrInvalidLifeInput)
		return
	}
	items, err := h.life.Events(c.Request.Context(), user, c.Query("domain"), m == "true")
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}
func (h *Handler) GetLifeEvent(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	item, err := h.life.Event(c.Request.Context(), user, id)
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": item})
}
func (h *Handler) CreateLifeEvent(c *gin.Context)         { h.saveLifeEvent(c, false, false) }
func (h *Handler) UpdateLifeEvent(c *gin.Context)         { h.saveLifeEvent(c, true, false) }
func (h *Handler) ConvertInboxToLifeEvent(c *gin.Context) { h.saveLifeEvent(c, false, true) }
func (h *Handler) saveLifeEvent(c *gin.Context, update, inbox bool) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	var id uint
	var input service.LifeEventInput
	if c.ShouldBindJSON(&input) != nil {
		lifeError(c, service.ErrInvalidLifeInput)
		return
	}
	if update || inbox {
		pathID, valid := projectPathID(c, "id")
		if !valid {
			return
		}
		if update {
			id = pathID
		} else {
			input.SourceType = "inbox"
			input.SourceID = &pathID
		}
	}
	item, err := h.life.SaveEvent(c.Request.Context(), user, id, input)
	if err != nil {
		lifeError(c, err)
		return
	}
	status := http.StatusCreated
	if update {
		status = 200
	}
	c.JSON(status, gin.H{"data": item})
}
func (h *Handler) DeleteLifeEvent(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.life.DeleteEvent(c.Request.Context(), user, id); err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"deleted": true}})
}
func (h *Handler) ListLifeGoals(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	items, err := h.life.Goals(c.Request.Context(), user)
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}
func (h *Handler) GetLifeGoal(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	goal, entries, events, err := h.life.Goal(c.Request.Context(), user, id)
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"goal": goal, "entries": entries, "events": events}})
}
func (h *Handler) CreateLifeGoal(c *gin.Context) { h.saveLifeGoal(c, false) }
func (h *Handler) UpdateLifeGoal(c *gin.Context) { h.saveLifeGoal(c, true) }
func (h *Handler) saveLifeGoal(c *gin.Context, update bool) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	var id uint
	if update {
		id, ok = projectPathID(c, "id")
		if !ok {
			return
		}
	}
	var input service.LifeGoalInput
	if c.ShouldBindJSON(&input) != nil {
		lifeError(c, service.ErrInvalidLifeInput)
		return
	}
	goal, err := h.life.SaveGoal(c.Request.Context(), user, id, input)
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": goal})
}
func (h *Handler) AddLifeGoalEntry(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input service.LifeEntryInput
	if c.ShouldBindJSON(&input) != nil {
		lifeError(c, service.ErrInvalidLifeInput)
		return
	}
	entry, err := h.life.AddEntry(c.Request.Context(), user, id, input)
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": entry})
}
func (h *Handler) GetLifeSource(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	source, err := h.life.Source(c.Request.Context(), user, id, c.Param("type"))
	if err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": source})
}
func (h *Handler) GraduateCourse(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.life.Graduate(c.Request.Context(), user, id); err != nil {
		lifeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"status": "completed"}})
}
