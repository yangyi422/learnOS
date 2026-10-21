package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

// dailyFocusDate accepts only a real YYYY-MM-DD calendar date; timestamps and
// loosely formatted strings are rejected with DAILY_FOCUS_DATE_INVALID.
func dailyFocusDate(date string) bool {
	if date == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", date)
	return err == nil && parsed.Format("2006-01-02") == date
}

func (h *Handler) GetDailyFocus(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.dailyFocus == nil {
		dailyFocusUnavailable(c)
		return
	}
	date := c.Query("date")
	if !dailyFocusDate(date) {
		dailyFocusError(c, service.ErrDailyFocusDateInvalid)
		return
	}
	scene := c.Query("context")
	if scene == "" {
		scene = "all"
	}
	day, err := h.dailyFocus.Day(c.Request.Context(), userID, date, scene)
	if err != nil {
		dailyFocusError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": service.DailyFocusDay{Date: date, Context: scene, Focuses: day}})
}

func (h *Handler) PutDailyFocus(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.dailyFocus == nil {
		dailyFocusUnavailable(c)
		return
	}
	date := c.Param("date")
	if !dailyFocusDate(date) {
		dailyFocusError(c, service.ErrDailyFocusDateInvalid)
		return
	}
	scene := c.Param("context")
	if scene != "work" && scene != "life" {
		dailyFocusError(c, service.ErrDailyFocusContextInvalid)
		return
	}
	var input struct {
		TaskID           json.RawMessage `json:"task_id"`
		ExpectedRevision *int            `json:"expected_revision"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		dailyFocusError(c, service.ErrDailyFocusInputInvalid)
		return
	}
	// task_id is required: a missing field is invalid input, while an explicit
	// JSON null clears the focus for that day and scene.
	rawTaskID := bytes.TrimSpace(input.TaskID)
	if len(rawTaskID) == 0 {
		dailyFocusError(c, service.ErrDailyFocusInputInvalid)
		return
	}
	expectedRevision := 0
	if input.ExpectedRevision != nil {
		expectedRevision = *input.ExpectedRevision
	}
	if expectedRevision < 0 {
		dailyFocusError(c, service.ErrDailyFocusInputInvalid)
		return
	}
	if bytes.Equal(rawTaskID, []byte("null")) {
		mutation, err := h.dailyFocus.Clear(c.Request.Context(), userID, date, scene, expectedRevision)
		if err != nil {
			dailyFocusError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": mutation})
		return
	}
	var taskID uint
	if err := json.Unmarshal(rawTaskID, &taskID); err != nil || taskID == 0 {
		dailyFocusError(c, service.ErrDailyFocusInputInvalid)
		return
	}
	mutation, err := h.dailyFocus.Set(c.Request.Context(), userID, date, scene, taskID, expectedRevision)
	if err != nil {
		dailyFocusError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mutation})
}

func dailyFocusUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "DAILY_FOCUS_UNAVAILABLE", "message": "今日焦点暂时无法读取，请稍后重试。", "retryable": true}})
}

// dailyFocusError mirrors the existing error envelope. Candidate failures use a
// single message so another user's task existence is never revealed.
func dailyFocusError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDailyFocusDateInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DAILY_FOCUS_DATE_INVALID", "message": "日期格式无效，请使用 YYYY-MM-DD。", "retryable": false}})
	case errors.Is(err, service.ErrDailyFocusContextInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DAILY_FOCUS_CONTEXT_INVALID", "message": "场景无效，仅支持 work / life。", "retryable": false}})
	case errors.Is(err, service.ErrDailyFocusInputInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DAILY_FOCUS_INPUT_INVALID", "message": "请求内容无效，请检查焦点任务与版本号。", "retryable": false}})
	case errors.Is(err, service.ErrDailyFocusTaskInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DAILY_FOCUS_TASK_INVALID", "message": "该任务不可用作当前场景的今日焦点。", "retryable": false}})
	case errors.Is(err, service.ErrDailyFocusConflict):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "DAILY_FOCUS_CONFLICT", "message": "今日焦点已被其他操作更新，请刷新后重试。", "retryable": false}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DAILY_FOCUS_UNAVAILABLE", "message": "今日焦点暂时无法读取，请稍后重试。", "retryable": true}})
	}
}
