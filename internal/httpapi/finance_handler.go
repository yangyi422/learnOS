package httpapi

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func financeError(c *gin.Context, e error) {
	status, msg := 500, "财务数据暂时无法读取或保存，请重试。"
	if errors.Is(e, service.ErrFinanceInput) {
		status, msg = 400, "请检查文件、来源账本、完整导出日期范围和确认选项。"
	} else if errors.Is(e, repository.ErrFinanceConflict) {
		status, msg = 409, "数据已变化或预览已过期，请重新预览；撤销需从最近的有效批次开始。"
	} else if errors.Is(e, gorm.ErrRecordNotFound) {
		status, msg = 404, "账本、交易或批次不存在，或不属于当前用户。"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": "FINANCE_REQUEST_FAILED", "message": msg, "retryable": status >= 500}})
}
func financeData(c *gin.Context, data interface{}, err error) {
	if err != nil {
		financeError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": data})
}
func (h *Handler) FinanceLedgers(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	a, e := h.finance.Ledgers(c.Request.Context(), u)
	financeData(c, a, e)
}
func (h *Handler) PreviewFinance(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.FinanceMaxBytes+65536)
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	if e := c.Request.ParseMultipartForm(service.FinanceMaxBytes + 65536); e != nil {
		financeError(c, service.ErrFinanceInput)
		return
	}
	file, header, e := c.Request.FormFile("file")
	if e != nil {
		financeError(c, service.ErrFinanceInput)
		return
	}
	defer file.Close()
	if header.Size > service.FinanceMaxBytes || strings.ToLower(filepath.Ext(header.Filename)) != ".csv" {
		financeError(c, service.ErrFinanceInput)
		return
	}
	data, e := io.ReadAll(io.LimitReader(file, service.FinanceMaxBytes+1))
	if e != nil || len(data) > service.FinanceMaxBytes {
		financeError(c, service.ErrFinanceInput)
		return
	}
	complete := c.PostForm("complete_scope")
	if complete != "" && complete != "true" && complete != "false" {
		financeError(c, service.ErrFinanceInput)
		return
	}
	a, e := h.finance.Preview(c.Request.Context(), u, service.FinancePreviewInput{LedgerName: c.PostForm("ledger_name"), Filename: filepath.Base(header.Filename), From: c.PostForm("scope_from"), To: c.PostForm("scope_to"), Complete: complete == "true"}, data)
	financeData(c, a, e)
}
func (h *Handler) ImportFinance(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var in struct {
		Remove  bool `json:"remove_missing"`
		Confirm bool `json:"confirm_missing"`
		Stable  bool `json:"confirm_stable_ids"`
	}
	if c.ShouldBindJSON(&in) != nil {
		financeError(c, service.ErrFinanceInput)
		return
	}
	a, e := h.finance.Import(c.Request.Context(), u, id, in.Remove, in.Confirm, in.Stable)
	financeData(c, a, e)
}
func (h *Handler) UndoFinance(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	a, e := h.finance.Undo(c.Request.Context(), u, id)
	financeData(c, a, e)
}
func financeFilter(c *gin.Context) (service.FinanceFilter, error) {
	f := service.FinanceFilter{From: c.Query("from"), To: c.Query("to"), Currency: c.Query("currency"), Category: c.Query("category"), Subcategory: c.Query("subcategory"), Tag: c.Query("tag"), Kind: c.Query("kind"), Keyword: c.Query("keyword"), Page: 1, PageSize: 30}
	for key, dest := range map[string]*int{"page": &f.Page, "page_size": &f.PageSize} {
		if raw := c.Query(key); raw != "" {
			v, e := strconv.Atoi(raw)
			if e != nil || v > 1000000 {
				return f, service.ErrFinanceInput
			}
			*dest = v
		}
	}
	f.CategoryEmpty = c.Query("category_empty") == "true"
	f.SubcategoryEmpty = c.Query("subcategory_empty") == "true"
	return f, f.Validate()
}
func (h *Handler) FinanceSummary(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "ledger")
	if !ok {
		return
	}
	f, e := financeFilter(c)
	if e != nil {
		financeError(c, e)
		return
	}
	a, e := h.finance.Summary(c.Request.Context(), u, id, f)
	financeData(c, a, e)
}
func (h *Handler) FinanceTransactions(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "ledger")
	if !ok {
		return
	}
	f, e := financeFilter(c)
	if e != nil {
		financeError(c, e)
		return
	}
	a, e := h.finance.Transactions(c.Request.Context(), u, id, f)
	financeData(c, a, e)
}
func (h *Handler) FinanceTransaction(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	l, ok := projectPathID(c, "ledger")
	if !ok {
		return
	}
	id, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	a, e := h.finance.Transaction(c.Request.Context(), u, l, id)
	financeData(c, a, e)
}
func (h *Handler) FinanceBatches(c *gin.Context) {
	u, ok := projectUserID(c)
	if !ok {
		return
	}
	l, ok := projectPathID(c, "ledger")
	if !ok {
		return
	}
	a, e := h.finance.Batches(c.Request.Context(), u, l)
	financeData(c, a, e)
}
