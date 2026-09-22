package model

import "time"

type CourseStatus string

const (
	CourseStatusInitializing CourseStatus = "initializing"
	CourseStatusLearning     CourseStatus = "learning"
	CourseStatusPaused       CourseStatus = "paused"
	CourseStatusCompleted    CourseStatus = "completed"
)

type Course struct {
	ID              uint         `json:"id" gorm:"primaryKey"`
	UserID          uint         `json:"user_id" gorm:"not null;index"`
	Name            string       `json:"name" gorm:"size:120;not null"`
	Description     string       `json:"description" gorm:"type:text"`
	Goal            string       `json:"goal" gorm:"type:text"`
	Status          CourseStatus `json:"status" gorm:"size:32;not null;index"`
	Progress        int          `json:"progress" gorm:"not null;default:0"`
	CurrentUnit     string       `json:"current_unit" gorm:"size:160"`
	CurrentUnitID   *uint        `json:"current_unit_id" gorm:"index"`
	CurrentLessonID *uint        `json:"current_lesson_id" gorm:"index"`
	LastStudiedAt   *time.Time   `json:"last_studied_at"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}
