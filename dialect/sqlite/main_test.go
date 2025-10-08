package sqlite

import (
	"testing"
)

func TestCanUseInTransaction(t *testing.T) {
	tests := []struct {
		sql      string
		expected bool
	}{
		{"CREATE TABLE test (id INT);", true},
		{"DROP TABLE test;", true},
		{"ALTER TABLE test ADD COLUMN name TEXT;", true},
		{"VACUUM;", false},
		{"PRAGMA user_version = 1;", true},
	}

	for _, tt := range tests {
		result := canUseInTransaction(tt.sql)
		if result != tt.expected {
			t.Errorf("SQLite3: sql=%q, expected=%v, got=%v", tt.sql, tt.expected, result)
		}
	}
}
