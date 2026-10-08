package repository

import (
	"context"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

func (r *LearningRepository) LessonConversation(ctx context.Context, courseID, lessonID uint) ([]model.LearningTurn, error) {
	var turns []model.LearningTurn
	err := r.db.WithContext(ctx).Where("course_id = ? AND lesson_id = ?", courseID, lessonID).Order("id ASC").Find(&turns).Error
	return turns, err
}
func (r *LearningRepository) CreateConversation(ctx context.Context, turn *model.LearningTurn) error {
	return r.db.WithContext(ctx).Create(turn).Error
}
func (r *LearningRepository) SaveConversationReply(ctx context.Context, turn *model.LearningTurn, evidence *model.CognitiveEvidence, run *model.AIEvaluationRun) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(turn).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Course{}).Where("id = ?", turn.CourseID).Update("last_studied_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		if evidence != nil {
			evidence.LearningTurnID = turn.ID
			if err := tx.Create(evidence).Error; err != nil {
				return err
			}
		}
		return tx.Create(run).Error
	})
}

// AdvanceLesson uses the old lesson ID as a compare-and-swap token. A replay
// cannot finish the newly selected lesson. It never writes mastery records.
func (r *LearningRepository) AdvanceLesson(ctx context.Context, courseID, lessonID uint, status model.LessonStatus) (*model.Lesson, error) {
	var next *model.Lesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var course model.Course
		if err := tx.First(&course, courseID).Error; err != nil {
			return err
		}
		var lesson model.Lesson
		if err := tx.Where("id = ? AND course_id = ?", lessonID, courseID).First(&lesson).Error; err != nil {
			return err
		}
		if lesson.Status == model.LessonStatusCompleted || lesson.Status == model.LessonStatusSkipped {
			if lesson.Status != status {
				return gorm.ErrInvalidData
			}
			if course.CurrentLessonID != nil && *course.CurrentLessonID != lessonID {
				var current model.Lesson
				if err := tx.First(&current, *course.CurrentLessonID).Error; err != nil {
					return err
				}
				next = &current
			}
			if course.CurrentLessonID == nil {
				return gorm.ErrInvalidData
			}
			if *course.CurrentLessonID != lessonID {
				return nil
			}
		}
		if course.CurrentLessonID == nil || *course.CurrentLessonID != lessonID {
			return gorm.ErrInvalidData
		}
		if err := tx.Model(&lesson).Update("status", status).Error; err != nil {
			return err
		}
		var candidates []model.Lesson
		if err := tx.Joins("JOIN course_units ON course_units.id = lessons.unit_id").Where("lessons.course_id = ?", courseID).Order("course_units.sort_order, lessons.sort_order, lessons.id").Find(&candidates).Error; err != nil {
			return err
		}
		currentIndex := -1
		for i, candidate := range candidates {
			if candidate.ID == lessonID {
				currentIndex = i
			}
		}
		ordered := append(append([]model.Lesson{}, candidates[currentIndex+1:]...), candidates[:currentIndex+1]...)
		candidates = nil
		for _, candidate := range ordered {
			if candidate.ID != lessonID && candidate.Status != model.LessonStatusCompleted && candidate.Status != model.LessonStatusSkipped {
				candidates = append(candidates, candidate)
			}
		}
		var total, done int64
		if err := tx.Model(&model.Lesson{}).Where("course_id = ?", courseID).Count(&total).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Lesson{}).Where("course_id = ? AND status IN ?", courseID, []string{"completed", "skipped"}).Count(&done).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"last_studied_at": time.Now().UTC()}
		if total > 0 {
			updates["progress"] = int(done * 100 / total)
		}
		if len(candidates) > 0 {
			next = &candidates[0]
			var unit model.CourseUnit
			if err := tx.First(&unit, next.UnitID).Error; err != nil {
				return err
			}
			updates["current_lesson_id"] = next.ID
			updates["current_unit_id"] = next.UnitID
			updates["current_unit"] = unit.Title
		} else {
			next = &lesson
			next.Status = status
		}
		return tx.Model(&course).Updates(updates).Error
	})
	return next, err
}

func (r *LearningRepository) LessonPosition(ctx context.Context, courseID, lessonID uint) (int, int, bool, error) {
	var lessons []model.Lesson
	err := r.db.WithContext(ctx).Joins("JOIN course_units ON course_units.id = lessons.unit_id").Where("lessons.course_id = ?", courseID).Order("course_units.sort_order, lessons.sort_order, lessons.id").Find(&lessons).Error
	position := 0
	hasNext := false
	for i, lesson := range lessons {
		if lesson.ID != lessonID && lesson.Status != model.LessonStatusCompleted && lesson.Status != model.LessonStatusSkipped {
			hasNext = true
		}
		if lesson.ID == lessonID {
			position = i + 1
		}
	}
	return position, len(lessons), hasNext, err
}
