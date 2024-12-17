package utils

import (
	"errors"
	"regexp"
)

var tableNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateTableName ensures the table name is safe for SQL use
func ValidateTableName(tableName string) error {
	if !tableNameRegex.MatchString(tableName) {
		return errors.New("invalid table name")
	}
	return nil
}
