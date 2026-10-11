package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"io"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
	"learnos/internal/repository"
	"path/filepath"
	"strings"
	"testing"
)

func libraryCollectionInput(name, template string) LibraryCollectionInput {
	return LibraryCollectionInput{Name: name, Template: template, DisplayMode: "wiki"}
}
func TestLibraryLifecycleSearchLinksExportAndRestore(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(dir, "library.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn, _ := db.DB(); conn.Close() })
	s := NewLibraryService(repository.NewLibraryRepository(db))
	ctx := context.Background()
	wiki, err := s.SaveCollection(ctx, 1, 0, libraryCollectionInput("合成游戏专题", "wiki"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Detail(ctx, 1, wiki.ID)
	if err != nil || loaded.Collection.CreatedAt.IsZero() || loaded.Collection.UpdatedAt.IsZero() {
		t.Fatal("missing persisted timestamps", loaded, err)
	}
	recipe, err := s.SaveCollection(ctx, 1, 0, libraryCollectionInput("合成菜谱", "recipe"))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.SaveCollection(ctx, 2, 0, libraryCollectionInput("私人", "general"))
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, 1)
	if err != nil || len(list) != 2 {
		t.Fatal(list, err)
	}
	section, err := s.SaveSection(ctx, 1, wiki.ID, 0, "章节")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.SaveSection(ctx, 1, recipe.ID, 0, "其他")
	if err != nil {
		t.Fatal(err)
	}
	in := LibraryPageInput{Title: "第四章攻略", Body: "## 实战\n\n中文正文，100%_准确\n\n|角色|建议|\n|---|---|\n|甲|乙|\n- [x] 已验证", SectionID: &section.ID, Tags: []string{"困难", "困难"}, Sources: []model.LibrarySource{{URL: "https://example.com/guide", Name: "外部来源", Note: "自己整理"}}}
	page, err := s.SavePage(ctx, 1, wiki.ID, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tags) != 1 || page.Sources[0].Note != "自己整理" {
		t.Fatal(page)
	}
	root, err := s.SavePage(ctx, 1, wiki.ID, 0, LibraryPageInput{Title: "根目录", Body: "[攻略](/explore/library/" + strconvUint(wiki.ID) + "/pages/" + strconvUint(page.ID) + ")"})
	if err != nil {
		t.Fatal(err)
	}
	dish, err := s.SavePage(ctx, 1, recipe.ID, 0, LibraryPageInput{Title: "合成菜品", Body: "## 食材\n\n练习数据"})
	if err != nil || dish.SectionID != nil {
		t.Fatal(err, dish)
	}
	for _, q := range []string{"第四章", "中文正文", "困难", "100%_准确"} {
		result, e := s.Pages(ctx, 1, wiki.ID, q, nil, false, 1)
		if e != nil || result.Total != 1 || result.Items[0].ID != page.ID {
			t.Fatal(q, result, e)
		}
	}
	empty, e := s.Pages(ctx, 1, recipe.ID, "第四章", nil, false, 1)
	if e != nil || empty.Total != 0 {
		t.Fatal(empty, e)
	}
	if _, err = s.Page(ctx, 2, wiki.ID, page.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("foreign read", err)
	}
	if _, err = s.Page(ctx, 1, foreign.ID, page.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	in.SectionID = &other.ID
	if _, err = s.SavePage(ctx, 1, wiki.ID, page.ID, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross collection section", err)
	}
	in.SectionID = &section.ID
	in.Version = page.Version
	in.Title = "改名后的攻略"
	in.Body += "\n更新内容"
	page, err = s.SavePage(ctx, 1, wiki.ID, page.ID, in)
	if err != nil || page.Version != 2 {
		t.Fatal(page, err)
	}
	if _, err = s.SavePage(ctx, 1, wiki.ID, page.ID, in); !errors.Is(err, repository.ErrLibraryConflict) {
		t.Fatal("lost update", err)
	}
	if err = s.DeleteSection(ctx, 1, wiki.ID, section.ID, false); !errors.Is(err, repository.ErrLibrarySectionNotEmpty) {
		t.Fatal(err)
	}
	if err = s.DeleteSection(ctx, 1, wiki.ID, section.ID, true); err != nil {
		t.Fatal(err)
	}
	moved, err := s.Page(ctx, 1, wiki.ID, page.ID)
	if err != nil || moved.SectionID != nil || moved.ID != page.ID || moved.Body != page.Body || moved.Version != page.Version+1 {
		t.Fatal(moved, err)
	}
	links, e := s.LinkStatus(ctx, 1, wiki.ID, []uint{page.ID, root.ID, dish.ID, 999})
	if e != nil || !links[page.ID] || links[dish.ID] || links[999] {
		t.Fatal(links, e)
	}
	updated := libraryCollectionInput(wiki.Name, "recipe")
	updated.Version = wiki.Version
	updated.DisplayMode = "list"
	wiki, err = s.SaveCollection(ctx, 1, wiki.ID, updated)
	if err != nil {
		t.Fatal(err)
	}
	zipdata, err := s.Export(ctx, 1, wiki.ID)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(zipdata), int64(len(zipdata)))
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, f := range reader.File {
		r, _ := f.Open()
		body, _ := io.ReadAll(r)
		r.Close()
		contents[f.Name] = string(body)
	}
	if !strings.Contains(contents["pages/"+strconvUint(root.ID)+".md"], "("+strconvUint(page.ID)+".md)") || !strings.Contains(contents["pages/"+strconvUint(page.ID)+".md"], "https://example.com/guide") || !strings.Contains(contents["metadata.json"], "更新内容") {
		t.Fatal("incomplete export", contents)
	}
	backup, err := database.CreateBackup(ctx, db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteCollection(ctx, 1, wiki.ID, wiki.Version); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.LibraryPage{}).Where("collection_id = ?", wiki.ID).Count(&count)
	if count != 0 {
		t.Fatal("dangling pages")
	}
	if _, err = s.Page(ctx, 1, recipe.ID, dish.ID); err != nil {
		t.Fatal("other collection changed", err)
	}
	conn, _ := db.DB()
	conn.Close()
	if _, err = RestoreDatabase(cfg.DatabasePath, backup.Path, cfg.BackupDir, 5); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s = NewLibraryService(repository.NewLibraryRepository(db))
	if restored, e := s.Page(ctx, 1, wiki.ID, page.ID); e != nil || restored.Body != moved.Body {
		t.Fatal("restore", restored, e)
	}
	snapshot, err := NewExportService(db).JSON(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]json.RawMessage
	if json.Unmarshal(snapshot, &exported) != nil || len(exported["library_pages"]) == 0 {
		t.Fatal("missing global export")
	}
	if err = s.DeletePage(ctx, 1, wiki.ID, page.ID, moved.Version); err != nil {
		t.Fatal(err)
	}
	links, err = s.LinkStatus(ctx, 1, wiki.ID, []uint{page.ID})
	if err != nil || links[page.ID] {
		t.Fatal("deleted link", links, err)
	}
}
func strconvUint(id uint) string { return fmt.Sprint(id) }
func TestLibraryValidationPaginationAndAtomicDelete(t *testing.T) {
	_, db := lifeTestService(t)
	s := NewLibraryService(repository.NewLibraryRepository(db))
	ctx := context.Background()
	col, err := s.SaveCollection(ctx, 1, 0, libraryCollectionInput("批量", "general"))
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []LibraryPageInput{{Title: "", Body: "x"}, {Title: "x", Sources: []model.LibrarySource{{URL: "javascript:alert(1)"}}}, {Title: "x", CoverURL: "data:image/svg+xml,x"}, {Title: "x", Body: strings.Repeat("中", 200001)}} {
		if _, err = s.SavePage(ctx, 1, col.ID, 0, in); !errors.Is(err, ErrLibraryInput) {
			t.Fatal(err)
		}
	}
	section, err := s.SaveSection(ctx, 1, col.ID, 0, "非空")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 125; i++ {
		p := model.LibraryPage{CollectionID: col.ID, Title: fmt.Sprintf("合成页面%d", i), Body: "中文", Tags: []string{}, Sources: []model.LibrarySource{}, SectionID: &section.ID, Version: 1}
		if err = db.Omit("Collection", "Section").Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Pages(ctx, 1, col.ID, "中文", nil, false, 1)
	last, e := s.Pages(ctx, 1, col.ID, "中文", nil, false, 3)
	if err != nil || e != nil || first.Total != 125 || len(first.Items) != 50 || len(last.Items) != 25 {
		t.Fatal(first, last, err, e)
	}
	db.Exec("CREATE TRIGGER fail_library_delete BEFORE DELETE ON library_sections BEGIN SELECT RAISE(ABORT,'synthetic failure'); END")
	if err = s.DeleteCollection(ctx, 1, col.ID, col.Version); err == nil {
		t.Fatal("expected rollback")
	}
	var count int64
	db.Model(&model.LibraryPage{}).Where("collection_id = ?", col.ID).Count(&count)
	if count != 125 {
		t.Fatal("partial delete", count)
	}
	db.Exec("DROP TRIGGER fail_library_delete")
	db.Exec("CREATE TRIGGER fail_library_move BEFORE DELETE ON library_sections BEGIN SELECT RAISE(ABORT,'synthetic failure'); END")
	if err = s.DeleteSection(ctx, 1, col.ID, section.ID, true); err == nil {
		t.Fatal("expected rollback")
	}
	db.Model(&model.LibraryPage{}).Where("section_id = ?", section.ID).Count(&count)
	if count != 125 {
		t.Fatal("partial move", count)
	}
}
