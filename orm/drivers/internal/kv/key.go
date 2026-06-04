package kv

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wxdqing/go-orm/orm"
)

func recordKey(table string, pk map[string]any) (string, error) {
	if table == "" {
		return "", fmt.Errorf("kv: table name is empty")
	}
	if len(pk) == 0 {
		return "", orm.ErrNoPrimaryKeySpecified
	}
	keys := make([]string, 0, len(pk))
	for k := range pk {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{table}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, pk[k]))
	}
	return strings.Join(parts, ":"), nil
}
