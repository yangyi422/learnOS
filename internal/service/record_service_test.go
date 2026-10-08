package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

func TestRecordConversionLifecycleAndIsolation(t *testing.T) {
	inbox, projects, db := inboxTestServices(t)
	records := NewRecordService(repository.NewRecordRepository(db))
	ctx := context.Background()
	project, err := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Project")})
	if err != nil {
		t.Fatal(err)
	}
	source, err := inbox.Create(ctx, 1, "技术决策原文")
	if err != nil {
		t.Fatal(err)
	}
	uri := "obsidian://open?vault=YY-Wiki&file=Projects%2FLearnOS%2F产品设计.md"
	item, record, err := inbox.ConvertToRecord(ctx, 1, source.ID, RecordInput{ProjectID: &project.ID, ExternalURL: uri, LinkName: "设计笔记"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Content != source.Content || item.Status != model.InboxStatusProcessed || item.ProcessedToType != model.InboxTargetRecord || *item.ProcessedToID != record.ID || record.Content != source.Content || *record.SourceInboxID != source.ID || record.ExternalURL != uri {
		t.Fatalf("bad conversion: %+v %+v", item, record)
	}
	if _, _, err = inbox.ConvertToRecord(ctx, 1, source.ID, RecordInput{}); !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
		t.Fatalf("repeat: %v", err)
	}
	if _, _, err = inbox.ConvertToTask(ctx, 1, source.ID, InboxConvertInput{ProjectID: project.ID}); !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
		t.Fatalf("cross conversion: %v", err)
	}
	changed, err := records.Save(ctx, 1, record.ID, RecordInput{Content: "修改后的记录", ExternalURL: "https://example.com"})
	if err != nil || changed.ProjectID != nil || changed.SourceInboxID == nil {
		t.Fatalf("save: %+v %v", changed, err)
	}
	original, _ := inbox.items.Find(ctx, 1, source.ID)
	if original.Content != source.Content {
		t.Fatal("original overwritten")
	}
	if _, err = records.Find(ctx, 2, record.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross user read")
	}
	if _, err = records.Save(ctx, 2, record.ID, RecordInput{Content: "attack"}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross user edit")
	}
	if err = records.Archive(ctx, 2, record.ID, true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross user archive")
	}
	if err = records.Delete(ctx, 2, record.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross user delete")
	}
	if err = records.Archive(ctx, 1, record.ID, true); err != nil {
		t.Fatal(err)
	}
	active, _ := records.List(ctx, 1, nil, false)
	archived, _ := records.List(ctx, 1, nil, true)
	if len(active) != 0 || len(archived) != 1 {
		t.Fatal("archive filter")
	}
	if err = records.Archive(ctx, 1, record.ID, false); err != nil {
		t.Fatal(err)
	}
	if err = inbox.Delete(ctx, 1, source.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = records.Find(ctx, 1, record.ID); err != nil {
		t.Fatal("Inbox deletion removed record")
	}
	if err = records.Delete(ctx, 1, record.ID); err != nil {
		t.Fatal(err)
	}
}
func TestRecordValidationRollbackAndProjectArchive(t *testing.T) {
	inbox, projects, db := inboxTestServices(t)
	records := NewRecordService(repository.NewRecordRepository(db))
	ctx := context.Background()
	project, _ := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Project")})
	item, _ := inbox.Create(ctx, 1, "retain")
	for _, input := range []RecordInput{{ExternalURL: "javascript:alert(1)"}, {ProjectID: ptr(uint(999))}, {ProjectID: ptr(uint(0))}} {
		if _, _, err := inbox.ConvertToRecord(ctx, 1, item.ID, input); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	original, _ := inbox.items.Find(ctx, 1, item.ID)
	if original.Status != model.InboxStatusInbox {
		t.Fatal("failed conversion claimed Inbox")
	}
	if err := db.Exec("CREATE TRIGGER reject_record_link BEFORE UPDATE OF processed_to_type ON inbox_items BEGIN SELECT RAISE(ABORT, 'forced failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := inbox.ConvertToRecord(ctx, 1, item.ID, RecordInput{}); err == nil {
		t.Fatal("expected transaction failure")
	}
	var count int64
	db.Model(&model.LightweightRecord{}).Count(&count)
	original, _ = inbox.items.Find(ctx, 1, item.ID)
	if count != 0 || original.Status != model.InboxStatusInbox || original.ProcessedAt != nil {
		t.Fatal("half-completed conversion")
	}
	db.Exec("DROP TRIGGER reject_record_link")
	record, err := records.Save(ctx, 1, 0, RecordInput{Content: "context", ProjectID: &project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err = projects.Update(ctx, 1, project.ID, ProjectPatch{Status: ptr("archived")}); err != nil {
		t.Fatal(err)
	}
	listed, err := records.List(ctx, 1, &project.ID, false)
	if err != nil || len(listed) != 1 || listed[0].ID != record.ID {
		t.Fatal("project archive hid record")
	}
	if err = db.Delete(&model.Project{}, project.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = records.Find(ctx, 1, record.ID); err != nil {
		t.Fatal("project deletion removed record")
	}
}
func TestInboxConcurrentRecordConversionAndCaptureRetries(t *testing.T) {
	inbox, _, db := inboxTestServices(t)
	ctx := context.Background()
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	item, err := inbox.CreateWithKey(ctx, 1, "retry content", "capture-test")
	if err != nil {
		t.Fatal(err)
	}
	again, err := inbox.CreateWithKey(ctx, 1, "retry content", "capture-test")
	if err != nil || item.ID != again.ID {
		t.Fatal("capture retry duplicated")
	}
	if _, err = inbox.CreateWithKey(ctx, 1, "different content", "capture-test"); !errors.Is(err, repository.ErrInboxCaptureConflict) {
		t.Fatal("capture key overwritten")
	}
	var wg sync.WaitGroup
	outcomes := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := inbox.ConvertToRecord(ctx, 1, item.ID, RecordInput{})
			outcomes <- err
		}()
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else if !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&model.LightweightRecord{}).Count(&count)
	if successes != 1 || count != 1 {
		t.Fatalf("successes=%d records=%d", successes, count)
	}
}

func TestInboxConcurrentTasksAndLateWriteRollback(t *testing.T) {
	inbox, projects, db := inboxTestServices(t)
	ctx := context.Background()
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	project, _ := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Project")})
	item, _ := inbox.Create(ctx, 1, "action")
	if err := db.Exec("CREATE TRIGGER reject_task_link BEFORE UPDATE OF processed_to_type ON inbox_items BEGIN SELECT RAISE(ABORT, 'forced failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := inbox.ConvertToTask(ctx, 1, item.ID, InboxConvertInput{ProjectID: project.ID}); err == nil {
		t.Fatal("expected late failure")
	}
	var count int64
	db.Model(&model.ProjectTask{}).Count(&count)
	unchanged, _ := inbox.items.Find(ctx, 1, item.ID)
	if count != 0 || unchanged.Status != model.InboxStatusInbox {
		t.Fatal("partial task conversion")
	}
	db.Exec("DROP TRIGGER reject_task_link")
	var wg sync.WaitGroup
	outcomes := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := inbox.ConvertToTask(ctx, 1, item.ID, InboxConvertInput{ProjectID: project.ID})
			outcomes <- err
		}()
	}
	wg.Wait()
	close(outcomes)
	success := 0
	for err := range outcomes {
		if err == nil {
			success++
		} else if !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
			t.Fatal(err)
		}
	}
	db.Model(&model.ProjectTask{}).Count(&count)
	if success != 1 || count != 1 {
		t.Fatalf("duplicate conversion: %d %d", success, count)
	}
}

func TestDirectRecordCreationRetriesAreIdempotent(t *testing.T) {
	_, _, db := inboxTestServices(t)
	ctx := context.Background()
	records := NewRecordService(repository.NewRecordRepository(db))
	input := RecordInput{Content: "context", CreationKey: "record-request-1"}
	first, err := records.Save(ctx, 1, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := records.Save(ctx, 1, 0, input)
	if err != nil || first.ID != repeated.ID {
		t.Fatalf("retry: %+v %v", repeated, err)
	}
	input.Content = "different"
	if _, err = records.Save(ctx, 1, 0, input); !errors.Is(err, repository.ErrInboxCaptureConflict) {
		t.Fatalf("changed retry: %v", err)
	}
	other, err := records.Save(ctx, 2, 0, input)
	if err != nil || other.ID == first.ID {
		t.Fatalf("isolated key: %+v %v", other, err)
	}
	var count int64
	db.Model(&model.LightweightRecord{}).Count(&count)
	if count != 2 {
		t.Fatalf("record count %d", count)
	}
}
