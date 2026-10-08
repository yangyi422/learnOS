package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInboxRecordsAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "records.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.InboxItem{}, &model.LightweightRecord{}); err != nil {
		t.Fatal(err)
	}
	projects := repository.NewProjectRepository(db)
	router := NewRouter(config.Config{Environment: "development"}, NewHandler(nil, nil, service.NewProjectService(projects), service.NewInboxService(repository.NewInboxRepository(db), projects), service.NewRecordService(repository.NewRecordRepository(db))), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}})
	call := func(method, path, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	call("POST", "/api/v1/projects", `{"title":"Project"}`, 201)
	call("POST", "/api/v1/inbox", `{"content":" "}`, 400)
	original := call("POST", "/api/v1/inbox", `{"content":"idea","capture_key":"retry-1"}`, 201)
	retry := call("POST", "/api/v1/inbox", `{"content":"idea","capture_key":"retry-1"}`, 201)
	var one, two struct{ Data model.InboxItem }
	json.Unmarshal([]byte(original), &one)
	json.Unmarshal([]byte(retry), &two)
	if one.Data.ID != two.Data.ID {
		t.Fatal("duplicate capture")
	}
	call("POST", "/api/v1/inbox/1/convert-to-record", `{"project_id":999}`, 404)
	call("POST", "/api/v1/inbox/1/convert-to-record", `{"external_url":"javascript:alert(1)"}`, 400)
	call("POST", "/api/v1/inbox/1/convert-to-record", `{"project_id":1,"external_url":"obsidian://open?vault=Wiki&file=Projects%2FLearnOS.md"}`, 201)
	call("POST", "/api/v1/inbox/1/convert-to-record", `{}`, 409)
	call("POST", "/api/v1/inbox/1/convert-to-task", `{"project_id":1}`, 409)
	if body := call("GET", "/api/v1/inbox?status=inbox", "", 200); strings.Contains(body, `"content":"idea"`) {
		t.Fatal("processed source still in Inbox")
	}
	if body := call("GET", "/api/v1/inbox?status=processed", "", 200); !strings.Contains(body, `"processed_to_type":"record"`) {
		t.Fatal("missing record target")
	}
	call("PUT", "/api/v1/records/1", `{"content":"edited","project_id":null,"external_url":"https://example.com","link_name":"reference"}`, 200)
	call("PUT", "/api/v1/records/1", `{"content":"edited","external_url":"file:///tmp/x"}`, 400)
	if body := call("GET", "/api/v1/records/1", "", 200); !strings.Contains(body, `"content":"edited"`) {
		t.Fatal("record not saved")
	}
	call("POST", "/api/v1/records/1/archive", `{"archived":true}`, 200)
	if body := call("GET", "/api/v1/records?status=active", "", 200); body != `{"data":[]}` {
		t.Fatal(body)
	}
	if body := call("GET", "/api/v1/records?status=archived", "", 200); !strings.Contains(body, "edited") {
		t.Fatal("archive inaccessible")
	}
	call("POST", "/api/v1/records/1/archive", `{"archived":false}`, 200)
	call("DELETE", "/api/v1/inbox/1", "", 200)
	call("GET", "/api/v1/records/1", "", 200)
	call("DELETE", "/api/v1/records/1", "", 200)
	call("GET", "/api/v1/records/1", "", 404)
	call("GET", "/api/v1/records?status=oops", "", 400)
	call("GET", "/api/v1/records?project_id=oops", "", 400)
}
