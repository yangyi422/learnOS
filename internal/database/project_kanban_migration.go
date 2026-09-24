package database

import (
	"fmt"

	"gorm.io/gorm"
)

// migrateProjectKanban is safe to retry after a partially completed startup.
// The old next-action flag is retained as historical data, but status is the
// sole source of truth for the new board.
func migrateProjectKanban(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE project_tasks SET status = CASE WHEN is_next_action = 1 THEN 'next' ELSE 'inbox' END WHERE status = 'todo'`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE project_tasks SET priority = 'normal' WHERE priority IS NULL OR priority = ''`).Error; err != nil {
			return err
		}
		// Legacy rows have no completion timestamp. updated_at is the best
		// available approximation and is documented as inferred history.
		if err := tx.Exec(`UPDATE project_tasks SET completed_at = updated_at WHERE status = 'done' AND completed_at IS NULL`).Error; err != nil {
			return err
		}
		type orderGroup struct {
			UserID   uint
			Status   string
			MaxOrder int64
		}
		var groups []orderGroup
		if err := tx.Raw(`SELECT p.user_id, t.status, COALESCE(MAX(t.sort_order), 0) AS max_order FROM project_tasks t JOIN projects p ON p.id = t.project_id GROUP BY p.user_id, t.status`).Scan(&groups).Error; err != nil {
			return err
		}
		next := map[string]int64{}
		for _, group := range groups {
			next[fmt.Sprintf("%d:%s", group.UserID, group.Status)] = group.MaxOrder
		}
		type unorderedTask struct {
			ID     uint
			UserID uint
			Status string
		}
		var unordered []unorderedTask
		if err := tx.Raw(`SELECT t.id, p.user_id, t.status FROM project_tasks t JOIN projects p ON p.id = t.project_id WHERE t.sort_order = 0 ORDER BY p.user_id, t.status, t.created_at, t.id`).Scan(&unordered).Error; err != nil {
			return err
		}
		for _, task := range unordered {
			key := fmt.Sprintf("%d:%s", task.UserID, task.Status)
			next[key] += 1024
			if err := tx.Exec(`UPDATE project_tasks SET sort_order = ? WHERE id = ? AND sort_order = 0`, next[key], task.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_project_tasks_board_order ON project_tasks(status, sort_order, id)`).Error; err != nil {
			return err
		}
		return tx.Exec(`DROP INDEX IF EXISTS idx_project_tasks_one_next_action`).Error
	})
}
