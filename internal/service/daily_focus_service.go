package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"
	"learnos/internal/repository"
)

// Daily focus surface errors. Handlers translate these into the documented
// DAILY_FOCUS_* error codes; candidate validation never reveals whether a
// foreign task exists.
var (
	ErrDailyFocusDateInvalid    = errors.New("invalid daily focus date")
	ErrDailyFocusContextInvalid = errors.New("invalid daily focus context")
	ErrDailyFocusInputInvalid   = errors.New("invalid daily focus input")
	ErrDailyFocusTaskInvalid    = errors.New("daily focus task is not available")
	ErrDailyFocusConflict       = errors.New("daily focus revision conflict")
)

// DailyFocusView is the API projection of one day/scene focus. State is derived
// on every read from the referenced task and its project; the stored row only
// remembers the user's choice.
type DailyFocusView struct {
	ID          uint       `json:"id"`
	Date        string     `json:"date"`
	Context     string     `json:"context"`
	TaskID      *uint      `json:"task_id"`
	Revision    int        `json:"revision"`
	State       string     `json:"state"`
	Title       string     `json:"title"`
	TaskStatus  string     `json:"task_status"`
	Priority    string     `json:"priority"`
	PlanDate    *string    `json:"plan_date"`
	DueDate     *string    `json:"due_date"`
	CompletedAt *time.Time `json:"completed_at"`
	// TaskUpdatedAt is the referenced task's real updated_at, used by the
	// client as expected_updated_at when completing the focus task. It is null
	// when the task is missing.
	TaskUpdatedAt  *time.Time `json:"task_updated_at"`
	ProjectID      uint       `json:"project_id"`
	ProjectTitle   string     `json:"project_title"`
	ProjectContext string     `json:"project_context"`
	ProjectStatus  string     `json:"project_status"`
	StateReason    string     `json:"state_reason"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// WorkspaceDailyFocus contains only the scenes the caller asked for. A key is
// present for every requested scene and is null when no focus is set.
type WorkspaceDailyFocus map[string]*DailyFocusView

type DailyFocusDay struct {
	Date    string              `json:"date"`
	Context string              `json:"context"`
	Focuses WorkspaceDailyFocus `json:"focuses"`
}

type DailyFocusMutation struct {
	Focus    *DailyFocusView `json:"focus"`
	Revision int             `json:"revision"`
}

type DailyFocusService struct {
	focuses *repository.DailyFocusRepository
}

func NewDailyFocusService(focuses *repository.DailyFocusRepository) *DailyFocusService {
	return &DailyFocusService{focuses: focuses}
}

const (
	dailyFocusWork = "work"
	dailyFocusLife = "life"
)

func validDailyFocusDate(date string) bool {
	if date == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", date)
	return err == nil && parsed.Format("2006-01-02") == date
}

func dailyFocusScenes(scene string) ([]string, error) {
	switch scene {
	case "all":
		return []string{dailyFocusWork, dailyFocusLife}, nil
	case dailyFocusWork, dailyFocusLife:
		return []string{scene}, nil
	default:
		return nil, ErrDailyFocusContextInvalid
	}
}

// Day returns the focus views of one date trimmed to the requested scene. The
// all scene returns both work and life; a concrete scene returns only itself.
func (s *DailyFocusService) Day(ctx context.Context, userID uint, date, scene string) (WorkspaceDailyFocus, error) {
	if !validDailyFocusDate(date) {
		return nil, ErrDailyFocusDateInvalid
	}
	scenes, err := dailyFocusScenes(scene)
	if err != nil {
		return nil, err
	}
	views := WorkspaceDailyFocus{}
	if s == nil || s.focuses == nil {
		for _, name := range scenes {
			views[name] = nil
		}
		return views, nil
	}
	rows, err := s.focuses.ListByDate(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	byScene := make(map[string]*model.DailyFocus, len(rows))
	for i := range rows {
		byScene[rows[i].Context] = &rows[i]
	}
	referenced := make([]*model.DailyFocus, 0, len(scenes))
	for _, name := range scenes {
		if row := byScene[name]; row != nil {
			referenced = append(referenced, row)
		}
	}
	references, err := s.taskReferences(ctx, userID, referenced)
	if err != nil {
		return nil, err
	}
	for _, name := range scenes {
		if row := byScene[name]; row != nil {
			views[name] = dailyFocusView(row, references)
		} else {
			views[name] = nil
		}
	}
	return views, nil
}

// Set points one day/scene focus at taskID after validating that the task can
// be a focus of that scene. The referenced task and its project are untouched.
func (s *DailyFocusService) Set(ctx context.Context, userID uint, date, scene string, taskID uint, expectedRevision int) (*DailyFocusMutation, error) {
	if !validDailyFocusDate(date) {
		return nil, ErrDailyFocusDateInvalid
	}
	if !oneOf(scene, dailyFocusWork, dailyFocusLife) {
		return nil, ErrDailyFocusContextInvalid
	}
	if taskID == 0 || expectedRevision < 0 {
		return nil, ErrDailyFocusInputInvalid
	}
	task, ok, err := s.focuses.Task(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	if !ok || !dailyFocusEligible(task, scene) {
		return nil, ErrDailyFocusTaskInvalid
	}
	row, err := s.focuses.Save(ctx, userID, date, scene, taskID, expectedRevision)
	if err != nil {
		if errors.Is(err, repository.ErrDailyFocusConflict) {
			return nil, ErrDailyFocusConflict
		}
		return nil, err
	}
	view := dailyFocusView(row, map[uint]repository.DailyFocusTask{taskID: *task})
	return &DailyFocusMutation{Focus: view, Revision: row.Revision}, nil
}

// Clear removes one day/scene focus. Clearing an absent focus is idempotent and
// reports revision 0.
func (s *DailyFocusService) Clear(ctx context.Context, userID uint, date, scene string, expectedRevision int) (*DailyFocusMutation, error) {
	if !validDailyFocusDate(date) {
		return nil, ErrDailyFocusDateInvalid
	}
	if !oneOf(scene, dailyFocusWork, dailyFocusLife) {
		return nil, ErrDailyFocusContextInvalid
	}
	if expectedRevision < 0 {
		return nil, ErrDailyFocusInputInvalid
	}
	if err := s.focuses.Clear(ctx, userID, date, scene, expectedRevision); err != nil {
		if errors.Is(err, repository.ErrDailyFocusConflict) {
			return nil, ErrDailyFocusConflict
		}
		return nil, err
	}
	return &DailyFocusMutation{Focus: nil, Revision: 0}, nil
}

func (s *DailyFocusService) taskReferences(ctx context.Context, userID uint, rows []*model.DailyFocus) (map[uint]repository.DailyFocusTask, error) {
	references := map[uint]repository.DailyFocusTask{}
	taskIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.TaskID != nil && *row.TaskID != 0 {
			taskIDs = append(taskIDs, *row.TaskID)
		}
	}
	if len(taskIDs) == 0 {
		return references, nil
	}
	return s.focuses.Tasks(ctx, userID, taskIDs)
}

func dailyFocusEligible(task *repository.DailyFocusTask, scene string) bool {
	return task.ProjectContext == scene &&
		task.ProjectStatus == "active" &&
		task.ArchivedAt == nil &&
		task.Status != "done"
}

// dailyFocusView derives the user-visible state. The check order is fixed:
// missing -> completed -> context_changed -> project_inactive -> archived -> ok.
func dailyFocusView(row *model.DailyFocus, references map[uint]repository.DailyFocusTask) *DailyFocusView {
	view := &DailyFocusView{
		ID:        row.ID,
		Date:      row.FocusDate,
		Context:   row.Context,
		TaskID:    row.TaskID,
		Revision:  row.Revision,
		UpdatedAt: row.UpdatedAt,
	}
	var task repository.DailyFocusTask
	found := false
	if row.TaskID != nil {
		task, found = references[*row.TaskID]
	}
	if !found {
		view.State = "missing"
		view.StateReason = "原任务已被删除。"
		return view
	}
	view.Title = task.Title
	view.TaskStatus = task.Status
	view.Priority = task.Priority
	view.PlanDate = task.PlanDate
	view.DueDate = task.DueDate
	view.CompletedAt = task.CompletedAt
	taskUpdatedAt := task.UpdatedAt
	view.TaskUpdatedAt = &taskUpdatedAt
	view.ProjectID = task.ProjectID
	view.ProjectTitle = task.ProjectTitle
	view.ProjectContext = task.ProjectContext
	view.ProjectStatus = task.ProjectStatus
	switch {
	case task.Status == "done":
		view.State = "completed"
	case task.ProjectContext != row.Context:
		view.State = "context_changed"
		view.StateReason = fmt.Sprintf("任务所属项目已改为「%s」场景。", dailyFocusSceneLabel(task.ProjectContext))
	case task.ProjectStatus != "active":
		view.State = "project_inactive"
		view.StateReason = dailyFocusProjectReason(task.ProjectStatus)
	case task.ArchivedAt != nil:
		view.State = "archived"
		view.StateReason = "任务已归档。"
	default:
		view.State = "ok"
	}
	return view
}

func dailyFocusSceneLabel(scene string) string {
	switch scene {
	case dailyFocusWork:
		return "工作"
	case dailyFocusLife:
		return "生活"
	default:
		return "未分类"
	}
}

func dailyFocusProjectReason(status string) string {
	switch status {
	case "paused":
		return "所属项目已暂停。"
	case "completed":
		return "所属项目已完成。"
	case "archived":
		return "所属项目已归档。"
	default:
		return "所属项目当前不可用。"
	}
}
