package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"learnos/internal/librarymedia"
	"learnos/internal/links"
	"learnos/internal/model"
	"learnos/internal/repository"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrLibraryInput = errors.New("invalid library input")

type LibraryService struct {
	repo   *repository.LibraryRepository
	images *LibraryImageService
}

func NewLibraryService(repo *repository.LibraryRepository, images ...*LibraryImageService) *LibraryService {
	s := &LibraryService{repo: repo}
	if len(images) > 0 {
		s.images = images[0]
	}
	return s
}

type LibraryCollectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Template    string `json:"template"`
	DisplayMode string `json:"display_mode"`
	CoverURL    string `json:"cover_url"`
	Version     uint   `json:"version"`
}
type LibraryPageInput struct {
	Title     string                `json:"title"`
	Body      string                `json:"body"`
	SectionID *uint                 `json:"section_id"`
	Tags      []string              `json:"tags"`
	Sources   []model.LibrarySource `json:"sources"`
	CoverURL  string                `json:"cover_url"`
	Version   uint                  `json:"version"`
}
type LibraryDetail struct {
	Collection model.LibraryCollection `json:"collection"`
	Sections   []model.LibrarySection  `json:"sections"`
}

func validLibraryText(value string, max int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(value) <= max
}
func libraryCover(value string) bool {
	return value == "" || (strings.HasPrefix(value, librarymedia.Prefix) && librarymedia.IDPattern.MatchString(strings.TrimPrefix(value, librarymedia.Prefix))) || (links.SafeExternal(value) && (strings.HasPrefix(strings.ToLower(value), "https://") || strings.HasPrefix(strings.ToLower(value), "http://")))
}
func (s *LibraryService) List(ctx context.Context, user uint) ([]repository.LibraryCollectionRow, error) {
	return s.repo.List(ctx, user)
}
func (s *LibraryService) Detail(ctx context.Context, user, id uint) (LibraryDetail, error) {
	c, sections, err := s.repo.Collection(ctx, user, id)
	return LibraryDetail{c, sections}, err
}
func (s *LibraryService) SaveCollection(ctx context.Context, user, id uint, in LibraryCollectionInput) (model.LibraryCollection, error) {
	if s.images != nil {
		librarymedia.Mu.Lock()
		defer librarymedia.Mu.Unlock()
		if err := s.images.ValidateReferences(ctx, user, in.CoverURL); err != nil {
			return model.LibraryCollection{}, err
		}
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	in.CoverURL = strings.TrimSpace(in.CoverURL)
	if in.Name == "" || !validLibraryText(in.Name, 120) || !validLibraryText(in.Description, 2000) || !validLibraryText(in.Category, 80) || !libraryCover(in.CoverURL) || (in.Template != "general" && in.Template != "wiki" && in.Template != "recipe") || (in.DisplayMode != "wiki" && in.DisplayMode != "list") {
		return model.LibraryCollection{}, ErrLibraryInput
	}
	row := model.LibraryCollection{UserID: user, Name: in.Name, Description: in.Description, Category: in.Category, Template: in.Template, DisplayMode: in.DisplayMode, CoverURL: in.CoverURL, Version: in.Version}
	if id == 0 {
		return s.repo.CreateCollection(ctx, row)
	}
	return s.repo.UpdateCollection(ctx, user, id, row)
}
func (s *LibraryService) DeleteCollection(ctx context.Context, user, id, version uint) error {
	return s.repo.DeleteCollection(ctx, user, id, version)
}
func (s *LibraryService) SaveSection(ctx context.Context, user, col, id uint, name string) (model.LibrarySection, error) {
	name = strings.TrimSpace(name)
	if name == "" || !validLibraryText(name, 120) {
		return model.LibrarySection{}, ErrLibraryInput
	}
	return s.repo.SaveSection(ctx, user, col, id, name)
}
func (s *LibraryService) DeleteSection(ctx context.Context, user, col, id uint, move bool) error {
	return s.repo.DeleteSection(ctx, user, col, id, move)
}
func (s *LibraryService) Page(ctx context.Context, user, col, id uint) (model.LibraryPage, error) {
	return s.repo.Page(ctx, user, col, id)
}
func (s *LibraryService) Pages(ctx context.Context, user, col uint, q string, section *uint, root bool, page int) (repository.LibraryPages, error) {
	if page < 1 || page > 100000 || !validLibraryText(q, 200) {
		return repository.LibraryPages{}, ErrLibraryInput
	}
	return s.repo.Pages(ctx, user, col, strings.TrimSpace(q), section, root, page)
}
func (s *LibraryService) SavePage(ctx context.Context, user, col, id uint, in LibraryPageInput) (model.LibraryPage, error) {
	if s.images != nil {
		librarymedia.Mu.Lock()
		defer librarymedia.Mu.Unlock()
		if err := s.images.ValidateReferences(ctx, user, in.Body, in.CoverURL); err != nil {
			return model.LibraryPage{}, err
		}
	}
	in.Title = strings.TrimSpace(in.Title)
	in.CoverURL = strings.TrimSpace(in.CoverURL)
	if in.Title == "" || !validLibraryText(in.Title, 200) || !validLibraryText(in.Body, 200000) || len(in.Body) > 1000000 || len(in.Tags) > 20 || len(in.Sources) > 30 || !libraryCover(in.CoverURL) || (in.SectionID != nil && *in.SectionID == 0) {
		return model.LibraryPage{}, ErrLibraryInput
	}
	tags := []string{}
	seen := map[string]bool{}
	for _, tag := range in.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || !validLibraryText(tag, 40) {
			return model.LibraryPage{}, ErrLibraryInput
		}
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	sources := []model.LibrarySource{}
	for _, source := range in.Sources {
		source.URL = strings.TrimSpace(source.URL)
		if source.URL == "" || !links.SafeExternal(source.URL) || !validLibraryText(source.Name, 200) || !validLibraryText(source.Note, 1000) {
			return model.LibraryPage{}, ErrLibraryInput
		}
		sources = append(sources, source)
	}
	row := model.LibraryPage{Title: in.Title, Body: in.Body, SectionID: in.SectionID, Tags: tags, Sources: sources, CoverURL: in.CoverURL, Version: in.Version}
	return s.repo.SavePage(ctx, user, col, id, row)
}
func (s *LibraryService) DeletePage(ctx context.Context, user, col, id, version uint) error {
	return s.repo.DeletePage(ctx, user, col, id, version)
}

// Export only canonical same-collection links. Stable numeric filenames are safe and rename-proof.
func (s *LibraryService) Export(ctx context.Context, user, col uint) ([]byte, error) {
	var buf bytes.Buffer
	if err := s.ExportTo(ctx, user, col, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (s *LibraryService) ExportTo(ctx context.Context, user, col uint, output io.Writer) error {
	librarymedia.Mu.RLock()
	defer librarymedia.Mu.RUnlock()

	c, sections, pages, err := s.repo.Export(ctx, user, col)
	if err != nil {
		return err
	}
	z := zip.NewWriter(output)
	defer z.Close()
	exportedImages := map[string]string{}
	exportImages := func(body string) (string, error) {
		var failure error
		body = librarymedia.References.ReplaceAllStringFunc(body, func(reference string) string {
			if failure != nil {
				return reference
			}
			id := strings.TrimPrefix(reference, librarymedia.Prefix)
			if name, ok := exportedImages[id]; ok {
				return "../assets/" + name
			}
			if s.images == nil {
				failure = ErrImageMissing
				return reference
			}
			row, data, err := s.images.ReadLocked(ctx, user, id)
			if err != nil {
				failure = ErrImageMissing
				return reference
			}
			w, err := z.Create("assets/" + row.StorageName)
			if err == nil {
				_, err = w.Write(data)
			}
			if err != nil {
				failure = err
				return reference
			}
			exportedImages[id] = row.StorageName
			return "../assets/" + row.StorageName
		})
		return body, failure
	}
	write := func(name, body string) error {
		w, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(body))
		return err
	}
	meta, _ := json.MarshalIndent(struct {
		Collection model.LibraryCollection `json:"collection"`
		Sections   []model.LibrarySection  `json:"sections"`
		Pages      []model.LibraryPage     `json:"pages"`
	}{c, sections, pages}, "", "  ")
	if err = write("metadata.json", string(meta)); err != nil {
		return err
	}
	toc := fmt.Sprintf("# %s\n\n%s\n\n", c.Name, c.Description)
	if c.CoverURL != "" {
		cover, e := exportImages(c.CoverURL)
		if e != nil {
			return e
		}
		cover = strings.TrimPrefix(cover, "../")
		toc += "![封面](" + cover + ")\n\n"
	}
	groupNames := map[uint]string{}
	for _, g := range sections {
		groupNames[g.ID] = g.Name
	}
	ids := map[uint]bool{}
	for _, p := range pages {
		ids[p.ID] = true
	}
	pattern := regexp.MustCompile(`/explore/library/` + strconv.FormatUint(uint64(col), 10) + `/pages/([0-9]+)`)
	for _, p := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		group := "专题根目录"
		if p.SectionID != nil {
			group = groupNames[*p.SectionID]
		}
		toc += fmt.Sprintf("- [%s](pages/%d.md) · %s\n", strings.NewReplacer("[", "\\[", "]", "\\]", "\n", " ").Replace(p.Title), p.ID, group)
		body := pattern.ReplaceAllStringFunc(p.Body, func(link string) string {
			matches := pattern.FindStringSubmatch(link)
			id, _ := strconv.ParseUint(matches[1], 10, 64)
			if ids[uint(id)] {
				return fmt.Sprintf("%d.md", id)
			}
			return link
		})
		body, err = exportImages(body)
		if err != nil {
			return err
		}
		if p.CoverURL != "" {
			cover, e := exportImages(p.CoverURL)
			if e != nil {
				return e
			}
			body = "![封面](" + cover + ")\n\n" + body
		}
		body = fmt.Sprintf("# %s\n\n%s\n\n## 外部资料来源\n\n", p.Title, body)
		for _, source := range p.Sources {
			body += fmt.Sprintf("- %s\n  <%s>\n  %s\n", source.Name, source.URL, source.Note)
		}
		body += fmt.Sprintf("\n---\n最近更新：%s\n", p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
		if err = write(fmt.Sprintf("pages/%d.md", p.ID), body); err != nil {
			return err
		}
	}
	if err = write("README.md", toc); err != nil {
		return err
	}
	if err = z.Close(); err != nil {
		return err
	}
	return nil
}

func (s *LibraryService) LinkStatus(ctx context.Context, user, col uint, ids []uint) (map[uint]bool, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, ErrLibraryInput
	}
	return s.repo.LinkStatus(ctx, user, col, ids)
}
