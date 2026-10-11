package httpapi

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/repository"
	"learnos/internal/service"
	"mime/multipart"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFinanceAPIUploadAuthenticationAndReadonly(t *testing.T) {
	db, e := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "api.db")})
	if e != nil {
		t.Fatal(e)
	}
	sql, _ := db.DB()
	defer sql.Close()
	s := service.NewFinanceService(repository.NewFinanceRepository(db))
	h := NewHandler(nil, nil, s)
	web := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	r := NewRouter(config.Config{Environment: "development"}, h, web)
	var csvData bytes.Buffer
	w := csv.NewWriter(&csvData)
	w.Write(service.FinanceHeaders)
	w.Write([]string{"id-a", "2026-01-02 12:00:00", "餐饮", "午餐", "支出", "12.34", "CNY", "", "", "私人备注,换行\n安全", "", "0", "0", "", "", "#日常#工作", "", ""})
	w.Flush()
	upload := func(data []byte, name string, want int) string {
		t.Helper()
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		f, _ := mw.CreateFormFile("file", name)
		f.Write(data)
		mw.WriteField("ledger_name", "个人账本")
		mw.Close()
		req := httptest.NewRequest("POST", "/api/v1/finance/preview", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatal(res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	invalid := upload([]byte("bad"), "x.csv", 200)
	if !strings.Contains(invalid, "errors") || strings.Contains(invalid, `"batch":{"`) {
		t.Fatal(invalid)
	}
	upload(csvData.Bytes(), "x.exe", 400)
	response := upload(csvData.Bytes(), "x.csv", 200)
	var p struct {
		Data service.FinancePreview `json:"data"`
	}
	if e = json.Unmarshal([]byte(response), &p); e != nil || p.Data.Batch == nil {
		t.Fatal(response, e)
	}
	call := func(method, path, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatal(method, path, res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	call("POST", "/api/v1/finance/batches/1/import", `{}`, 200)
	call("POST", "/api/v1/finance/batches/1/import", `{}`, 200)
	call("GET", "/api/v1/finance/ledgers/1/transactions?page=0", "", 400)
	call("GET", "/api/v1/finance/ledgers/1/summary?from=2026-02-30", "", 400)
	result := call("GET", "/api/v1/finance/ledgers/1/transactions/1", "", 200)
	if !strings.Contains(result, `"amount_minor":"1234"`) || !strings.Contains(result, `"raw"`) || !strings.Contains(result, "私人备注") {
		t.Fatal(result)
	}
	call("PUT", "/api/v1/finance/ledgers/1/transactions/1", `{"amount_minor":"0"}`, 404)
	if result = call("GET", "/api/v1/finance/ledgers/1/transactions?tag=日常", "", 200); !strings.Contains(result, `"total":1`) {
		t.Fatal(result)
	}
	call("POST", "/api/v1/finance/batches/1/undo", `{}`, 200)
	call("POST", "/api/v1/finance/batches/1/undo", `{}`, 200)
	// Authentication applies equally to uploads, statistics, raw detail and undo.
	protected := NewRouter(config.Config{Environment: "production"}, h, web)
	for _, test := range []struct{ method, path string }{{"GET", "/api/v1/finance/ledgers"}, {"POST", "/api/v1/finance/preview"}, {"GET", "/api/v1/finance/ledgers/1/summary"}, {"GET", "/api/v1/finance/ledgers/1/transactions/1"}, {"POST", "/api/v1/finance/batches/1/undo"}} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest(test.method, test.path, nil))
		if res.Code != 401 {
			t.Fatal("unprotected", test, res.Code)
		}
	}
	// Request size limit is enforced before CSV parsing, not just by the browser.
	body := io.LimitReader(strings.NewReader(strings.Repeat("x", service.FinanceMaxBytes+100000)), int64(service.FinanceMaxBytes+100000))
	req := httptest.NewRequest("POST", "/api/v1/finance/preview", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 400 {
		t.Fatal(res.Code)
	}
}
