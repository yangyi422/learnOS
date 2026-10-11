package model

import "time"

// Amounts use signed integer minor units; no floating point enters persistence.
type FinanceLedger struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"not null;uniqueIndex:idx_finance_ledger,priority:1"`
	Name      string    `json:"name" gorm:"not null;uniqueIndex:idx_finance_ledger,priority:2"`
	Revision  uint      `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type FinanceTransaction struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	LedgerID    uint      `json:"ledger_id" gorm:"not null;uniqueIndex:idx_finance_transaction,priority:1;index"`
	ExternalID  string    `json:"external_id" gorm:"not null;uniqueIndex:idx_finance_transaction,priority:2"`
	OccurredAt  string    `json:"occurred_at" gorm:"index"`
	Kind        string    `json:"kind" gorm:"index"`
	AmountMinor int64     `json:"amount_minor,string"`
	Currency    string    `json:"currency" gorm:"index"`
	Category    string    `json:"category"`
	Subcategory string    `json:"subcategory"`
	Note        string    `json:"note" gorm:"type:text"`
	TagsJSON    string    `json:"-" gorm:"type:text"`
	RawJSON     string    `json:"-" gorm:"type:text"`
	ContentHash string    `json:"-"`
	BatchID     uint      `json:"batch_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type FinanceBatch struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	LedgerID      uint       `json:"ledger_id" gorm:"not null;index"`
	Filename      string     `json:"filename"`
	FileHash      string     `json:"file_hash"`
	Status        string     `json:"status" gorm:"index"` // preview / imported / undone
	BaseRevision  uint       `json:"base_revision"`
	ScopeFrom     string     `json:"scope_from"`
	ScopeTo       string     `json:"scope_to"`
	CompleteScope bool       `json:"complete_scope"`
	RemoveMissing bool       `json:"remove_missing"`
	Added         int        `json:"added"`
	Updated       int        `json:"updated"`
	Unchanged     int        `json:"unchanged"`
	Missing       int        `json:"missing"`
	Removed       int        `json:"removed"`
	RowCount      int        `json:"row_count"`
	PayloadJSON   string     `json:"-" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at"`
	ImportedAt    *time.Time `json:"imported_at"`
	UndoneAt      *time.Time `json:"undone_at"`
}
type FinanceChange struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	BatchID       uint   `json:"batch_id" gorm:"not null;index"`
	TransactionID uint   `json:"transaction_id"`
	BeforeJSON    string `json:"before_json" gorm:"type:text"`
	AfterJSON     string `json:"after_json" gorm:"type:text"`
}
