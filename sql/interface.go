package sql

import (
	"encoding/json"
	"regexp"
	"strings"
)

// QueryBuilder defines the interface for SQL query builders
type QueryBuilder interface {
	// BuildQuery constructs an SQL query from HTTP request components
	BuildQuery(method string, table string, id string, relations []string, filters map[string]string, body json.RawMessage) (string, map[string]interface{}, error)

	// GetPlaceholder returns the parameter placeholder for the specific SQL dialect
	GetPlaceholder(index int) string

	// QuoteIdentifier returns a quoted identifier for the specific SQL dialect
	QuoteIdentifier(name string) string
}

var (
	allowedFunctions = []string{
		// math functions
		"abs", "avg", "ceil", "div", "exp", "floor", "gcd", "lcm", "ln", "log",
		"mod", "power", "round", "sign", "sqrt", "trunc", "max", "min", "sum",
		// date functions
		"date", "date_format", "date_part", "date_trunc", "extract", "hour",
		"minute", "month", "second", "utctimestamp", "weekofday", "year",
		"time", "datetime", "julianday", "unixepoch", "strftime",
		// string functions
		"bit_length", "chr", "char_length", "left", "length", "ord", "trim",
	}
	allowedFunctionExp = regexp.MustCompile(strings.Join(allowedFunctions, "|"))
	funcExp            = regexp.MustCompile(`(.*?)\(`)
	invalidIdentifier  = regexp.MustCompile("[ ;'\"]")
)
