package repository

import (
	"errors"
	"gorm.io/gorm"
	"learnos/internal/model"
	"time"
)

type ReviewSources struct {
	Version      uint
	Tasks        []model.ProjectTask
	Projects     []model.Project
	Courses      []model.Course
	Lessons      []model.Lesson
	Turns        []model.LearningTurn
	Questions    []model.ExplorationQuestion
	Events       []model.LifeEvent
	Entries      []model.LifeGoalEntry
	LessonTitles map[uint]string
	GoalTitles   map[uint]string
}

func CollectReviewSources(tx *gorm.DB, user uint, month string, start, end time.Time) (ReviewSources, error) {
	s := ReviewSources{LessonTitles: map[uint]string{}, GoalTitles: map[uint]string{}}
	var saved model.MonthlyReview
	err := tx.Where("user_id = ? AND month = ?", user, month).First(&saved).Error
	if err == nil {
		s.Version = saved.Version
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return s, err
	}
	dateStart, dateEnd := start.Format("2006-01-02"), end.Format("2006-01-02")
	queries := []struct {
		q    *gorm.DB
		dest interface{}
	}{
		{tx.Joins("JOIN projects p ON p.id = project_tasks.project_id").Where("p.user_id = ? AND project_tasks.status = 'done' AND project_tasks.actual_completed_at >= ? AND project_tasks.actual_completed_at < ?", user, start.UTC(), end.UTC()).Order("project_tasks.actual_completed_at DESC, project_tasks.id DESC"), &s.Tasks},
		{tx.Where("user_id = ? AND completed_at >= ? AND completed_at < ?", user, start.UTC(), end.UTC()), &s.Projects},
		{tx.Where("user_id = ? AND completed_at >= ? AND completed_at < ?", user, start.UTC(), end.UTC()), &s.Courses},
		{tx.Joins("JOIN courses c ON c.id = lessons.course_id").Where("c.user_id = ? AND lessons.status = 'completed' AND lessons.completed_at >= ? AND lessons.completed_at < ?", user, start.UTC(), end.UTC()), &s.Lessons},
		{tx.Joins("JOIN courses c ON c.id = learning_turns.course_id").Where("c.user_id = ? AND learning_turns.created_at >= ? AND learning_turns.created_at < ?", user, start.UTC(), end.UTC()).Order("learning_turns.created_at DESC,learning_turns.id DESC"), &s.Turns},
		{tx.Joins("JOIN courses c ON c.id = exploration_questions.course_id").Where("c.user_id = ? AND exploration_questions.created_at >= ? AND exploration_questions.created_at < ?", user, start.UTC(), end.UTC()), &s.Questions},
		{tx.Where("user_id = ? AND occurred_on >= ? AND occurred_on < ?", user, dateStart, dateEnd).Order("occurred_on DESC,id DESC"), &s.Events},
		{tx.Joins("JOIN life_goals g ON g.id = life_goal_entries.goal_id").Where("g.user_id = ? AND life_goal_entries.occurred_on >= ? AND life_goal_entries.occurred_on < ?", user, dateStart, dateEnd).Order("life_goal_entries.occurred_on DESC,life_goal_entries.id DESC"), &s.Entries},
	}
	for _, query := range queries {
		if err = query.q.Find(query.dest).Error; err != nil {
			return s, err
		}
	}
	ids := []uint{}
	for _, t := range s.Turns {
		ids = append(ids, t.LessonID)
	}
	var titles []model.Lesson
	if len(ids) > 0 {
		if err = tx.Select("lessons.id, lessons.title").Joins("JOIN courses c ON c.id = lessons.course_id").Where("c.user_id = ? AND lessons.id IN ?", user, ids).Find(&titles).Error; err != nil {
			return s, err
		}
		for _, l := range titles {
			s.LessonTitles[l.ID] = l.Title
		}
	}
	ids = nil
	for _, e := range s.Entries {
		ids = append(ids, e.GoalID)
	}
	var goals []model.LifeGoal
	if len(ids) > 0 {
		if err = tx.Select("id,title").Where("user_id = ? AND id IN ?", user, ids).Find(&goals).Error; err != nil {
			return s, err
		}
		for _, g := range goals {
			s.GoalTitles[g.ID] = g.Title
		}
	}
	return s, nil
}
func ReviewSourceAvailable(tx *gorm.DB, user uint, kind string, id uint) (bool, error) {
	var count int64
	var q *gorm.DB
	switch kind {
	case "task":
		q = tx.Model(&model.ProjectTask{}).Joins("JOIN projects p ON p.id=project_tasks.project_id").Where("project_tasks.id = ? AND p.user_id = ?", id, user)
	case "project":
		q = tx.Model(&model.Project{}).Where("id = ? AND user_id = ?", id, user)
	case "course":
		q = tx.Model(&model.Course{}).Where("id = ? AND user_id = ?", id, user)
	case "lesson":
		q = tx.Model(&model.Lesson{}).Joins("JOIN courses c ON c.id=lessons.course_id").Where("lessons.id = ? AND c.user_id = ?", id, user)
	case "learning_activity":
		q = tx.Model(&model.LearningTurn{}).Joins("JOIN courses c ON c.id=learning_turns.course_id").Where("learning_turns.id = ? AND c.user_id = ?", id, user)
	case "exploration":
		q = tx.Model(&model.ExplorationQuestion{}).Joins("JOIN courses c ON c.id=exploration_questions.course_id").Where("exploration_questions.id = ? AND c.user_id = ?", id, user)
	case "event":
		q = tx.Model(&model.LifeEvent{}).Where("id = ? AND user_id = ?", id, user)
	case "goal_entry":
		q = tx.Model(&model.LifeGoalEntry{}).Joins("JOIN life_goals g ON g.id=life_goal_entries.goal_id").Where("life_goal_entries.id = ? AND g.user_id = ?", id, user)
	case "ledger":
		q = tx.Model(&model.FinanceLedger{}).Where("id = ? AND user_id = ?", id, user)
	case "asset":
		q = tx.Model(&model.AssetSnapshot{}).Where("id = ? AND user_id = ?", id, user)
	default:
		return false, nil
	}
	err := q.Count(&count).Error
	return count > 0, err
}
