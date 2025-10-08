package postgres

import (
	"testing"
)

func TestPostgresCanUseInTransaction(t *testing.T) {
	tests := []struct {
		sql      string
		expected bool
	}{
		{"CREATE TABLE test (id INT);", true},
		{"DROP TABLE test;", true},
		{"ALTER TABLE test ADD COLUMN name TEXT;", true},
		{"VACUUM;", false},
		{"CLUSTER;", false},
		{"CREATE DATABASE sample;", false},
		{"DROP DATABASE sample;", false},
		{"CREATE TABLESPACE ts1 LOCATION '/tmp';", false},
		{"DROP TABLESPACE ts1;", false},
	}

	for _, tt := range tests {
		result := canUseInTransaction(tt.sql)
		if result != tt.expected {
			t.Errorf("Postgres: sql=%q, expected=%v, got=%v", tt.sql, tt.expected, result)
		}
	}
}
