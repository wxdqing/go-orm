package logger_ext

import (
	"context"
	"testing"
)

func TestNewDbLoggerOmitsQueryParameters(t *testing.T) {
	filter, ok := NewDbLogger().(interface {
		ParamsFilter(context.Context, string, ...interface{}) (string, []interface{})
	})
	if !ok {
		t.Fatal("database logger does not support parameter filtering")
	}
	sql, params := filter.ParamsFilter(context.Background(), "INSERT INTO account (credential) VALUES (?)", "credential-secret")
	if sql != "INSERT INTO account (credential) VALUES (?)" {
		t.Fatalf("SQL changed: %q", sql)
	}
	if len(params) != 0 {
		t.Fatalf("query parameters retained: %v", params)
	}
}
