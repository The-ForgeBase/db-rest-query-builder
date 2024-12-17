package utils

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type TypeConverter func(any) any

var (
	numericRegexp = regexp.MustCompile(`^(INT|FLOAT)\d+`)
	// Various data types
	// PG: https://www.postgresql.org/docs/current/datatype.html
	// MY: https://dev.mysql.com/doc/refman/8.0/en/data-types.html
	// SQLITE: https://www.sqlite.org/datatype3.html

	// the code below could be simplified by using regexp, but declare it in a
	// map should result in better performance in theory.
	Types = map[string]func() any{
		"TINYINT":     func() any { return new(sql.NullInt64) },
		"SMALLINT":    func() any { return new(sql.NullInt64) },
		"SMALLSERIAL": func() any { return new(sql.NullInt64) },
		"SERIAL":      func() any { return new(sql.NullInt64) },
		"INT":         func() any { return new(sql.NullInt64) },
		"INTEGER":     func() any { return new(sql.NullInt64) },
		"BIGINT":      func() any { return new(sql.NullInt64) },
		"BIGSERIAL":   func() any { return new(sql.NullInt64) },

		"DEC":              func() any { return new(sql.NullFloat64) },
		"DECIMAL":          func() any { return new(sql.NullFloat64) },
		"NUMERIC":          func() any { return new(sql.NullFloat64) },
		"FLOAT":            func() any { return new(sql.NullFloat64) },
		"REAL":             func() any { return new(sql.NullFloat64) },
		"DOUBLE":           func() any { return new(sql.NullFloat64) },
		"DOUBLE PRECISION": func() any { return new(sql.NullFloat64) },

		"BOOL":    func() any { return new(sql.NullBool) },
		"BOOLEAN": func() any { return new(sql.NullBool) },

		"JSON": func() any { return new(sql.NullString) },

		"CHAR":      func() any { return new(sql.NullString) },
		"VARCHAR":   func() any { return new(sql.NullString) },
		"NVARCHAR":  func() any { return new(sql.NullString) },
		"TEXT":      func() any { return new(sql.NullString) },
		"UUID":      func() any { return new(sql.NullString) },
		"ENUM":      func() any { return new(sql.NullString) },
		"BLOB":      func() any { return new(sql.NullString) },
		"BINARY":    func() any { return new(sql.NullString) },
		"XML":       func() any { return new(sql.NullString) },
		"DATE":      func() any { return new(sql.NullString) },
		"DATETIME":  func() any { return new(sql.NullString) },
		"TIMESTAMP": func() any { return new(sql.NullString) },
	}

	TypeConverters = map[string]TypeConverter{
		"TINYINT":     func(i any) any { return i.(*sql.NullInt64).Int64 },
		"SMALLINT":    func(i any) any { return i.(*sql.NullInt64).Int64 },
		"SMALLSERIAL": func(i any) any { return i.(*sql.NullInt64).Int64 },
		"SERIAL":      func(i any) any { return i.(*sql.NullInt64).Int64 },
		"INT":         func(i any) any { return i.(*sql.NullInt64).Int64 },
		"INTEGER":     func(i any) any { return i.(*sql.NullInt64).Int64 },
		"BIGINT":      func(i any) any { return i.(*sql.NullInt64).Int64 },
		"BIGSERIAL":   func(i any) any { return i.(*sql.NullInt64).Int64 },

		"DEC":              func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"DECIMAL":          func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"NUMERIC":          func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"FLOAT":            func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"REAL":             func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"DOUBLE":           func(i any) any { return i.(*sql.NullFloat64).Float64 },
		"DOUBLE PRECISION": func(i any) any { return i.(*sql.NullFloat64).Float64 },

		"BOOL":    func(i any) any { return i.(*sql.NullBool).Bool },
		"BOOLEAN": func(i any) any { return i.(*sql.NullBool).Bool },

		"CHAR":      func(i any) any { return i.(*sql.NullString).String },
		"VARCHAR":   func(i any) any { return i.(*sql.NullString).String },
		"NVARCHAR":  func(i any) any { return i.(*sql.NullString).String },
		"TEXT":      func(i any) any { return i.(*sql.NullString).String },
		"UUID":      func(i any) any { return i.(*sql.NullString).String },
		"ENUM":      func(i any) any { return i.(*sql.NullString).String },
		"BLOB":      func(i any) any { return i.(*sql.NullString).String },
		"BINARY":    func(i any) any { return i.(*sql.NullString).String },
		"XML":       func(i any) any { return i.(*sql.NullString).String },
		"DATE":      func(i any) any { return i.(*sql.NullString).String },
		"DATETIME":  func(i any) any { return i.(*sql.NullString).String },
		"TIMESTAMP": func(i any) any { return i.(*sql.NullString).String },

		"JSON": func(i any) any {
			rawData := i.(*sql.NullString).String
			if s, err := strconv.ParseFloat(rawData, 64); err == nil {
				return s
			}
			if s, err := strconv.ParseBool(rawData); err == nil {
				return s
			}
			return rawData
		},
	}

	Operators = map[string]string{
		"eq":   "=",
		"ne":   "<>",
		"gt":   ">",
		"gte":  ">=",
		"lt":   "<",
		"lte":  "<=",
		"is":   "IS",
		"like": "LIKE",
	}

	ReservedWords = map[string]struct{}{
		"select": {},
		"order":  {},
		"count":  {},
	}
)

type ReturnQuery struct {
	Query string
	Args  []any
}

// ParseQueryParam tries to convert a query parameter string to an appropriate type (int, float64, bool, or string)
func ParseQueryParam(value string) (interface{}, error) {
	// Check if it's a boolean
	if strings.ToLower(value) == "true" || strings.ToLower(value) == "false" {
		return strconv.ParseBool(value)
	}

	// Check if it's an integer
	if i, err := strconv.ParseInt(value, 0, 64); err == nil {
		// fmt.Println("Parsed int:", i)
		return int64(i), nil
	}

	// Check if it's a float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		fmt.Println("Parsed float:", f)
		return f, nil
	}

	// Default to string if it can't be parsed as int, float, or bool
	return value, nil
}
