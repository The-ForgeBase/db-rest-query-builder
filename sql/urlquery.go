package sql

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

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
	jsonPathFunc       = map[string]func(column string) (jsonPath, asName string){
		"postgres": buildPGJSONPath,
		"mysql":    buildMysqlJSONPath,
		"sqlite":   buildSqliteJSONPath,
	}
)

type URLQuery struct {
	values url.Values
	driver string
}

func NewURLQuery(values url.Values, driver string) *URLQuery {
	return &URLQuery{values, driver}
}

func (q *URLQuery) Set(key, value string) {
	q.values[key] = []string{value}
}

// SelectQuery return sql projection string
func (q *URLQuery) SelectQuery() (string, error) {
	selects := q.values["select"]
	if len(selects) == 0 {
		return "*", nil
	}

	selectVal := selects[0]
	if invalidIdentifier.MatchString(selectVal) {
		return "", errors.New("invalid character found")
	}

	columns := strings.Split(selectVal, ",")
	for i, c := range columns {
		// TODO: fail fast if there are duplicate column names
		column, err := q.buildColumn(c, true)
		if err != nil {
			return "", err
		}
		columns[i] = column
	}
	return strings.Join(columns, ","), nil
}

// OrderQuery returns sql order query string
func (q *URLQuery) OrderQuery() string {
	orders := q.values["order"]
	if len(orders) == 0 {
		return ""
	}
	if invalidIdentifier.MatchString(orders[0]) {
		// log.Warn("invalid character in order: ", orders[0])
		return ""
	}

	return strings.ReplaceAll(orders[0], ".", " ")
}

// WhereQuery returns the SQL WHERE clause and the associated arguments for the query.
// It processes the query parameters from the URL and constructs the appropriate SQL conditions.
func (q *URLQuery) WhereQuery(index uint) (newIndex uint, query string, args []any) {
	// Check if there are any query values. If not, return early with the current index and empty query and args.
	if len(q.values) == 0 {
		return index, "", nil
	}

	// Create a strings.Builder to efficiently build the SQL query string.
	var queryBuilder strings.Builder
	// Initialize args slice to hold the values for the SQL query placeholders.
	args = make([]any, 0, len(q.values))
	// A flag to determine if this is the first condition being added to the query.
	first := true

	// Iterate over each key-value pair in the URL query values.
	for k, v := range q.values {
		// Skip reserved words that should not be included in the SQL query.
		if _, ok := ReservedWords[k]; ok {
			continue
		}
		// Iterate over each value associated with the key.
		for _, vv := range v {
			// Split the value by '.' to separate the operator from the actual value.
			vals := strings.Split(vv, ".")
			// Ensure that exactly two parts are obtained (operator and value).
			if len(vals) != 2 {
				continue
			}
			// Assign the operator and value from the split.
			op, val := vals[0], vals[1]
			// Check if the operator is valid by looking it up in the Operators map.
			operator, ok := Operators[op]
			if !ok {
				// Log a warning if the operator is unsupported and continue to the next value.
				// log.Warnf("unsupported op: %s", op)
				continue
			}

			// If this is not the first condition, prepend ' AND ' to the query.
			if !first {
				queryBuilder.WriteString(" AND ")
			}

			// Build the SQL column name using the key and append it to the query.
			column, err := q.buildColumn(k, false)
			if err != nil {
				return index, "", nil
			}
			queryBuilder.WriteString(column)

			// Handle the 'in' operator specifically.
			if op == "in" {
				// Remove parentheses and split the values by comma.
				vals := strings.Split(strings.Trim(strings.Trim(val, ")"), "("), ",")
				// Create placeholders for each value and append them to the args.
				placeholders := make([]string, len(vals))
				for i, v := range vals {
					placeholders[i] = "?"
					args = append(args, v)
					index++
				}
				// Append the 'IN' clause to the query with the placeholders.
				queryBuilder.WriteString(fmt.Sprintf(" IN (%s)", strings.Join(placeholders, ",")))
			} else if op == "is" {
				// Handle the 'is' operator for boolean and null checks.
				if strings.EqualFold(val, "true") || strings.EqualFold(val, "false") ||
					strings.EqualFold(val, "null") {
					queryBuilder.WriteString(operator)
					queryBuilder.WriteString(val)
				} else {
					// Log a warning for unsupported values for the 'is' operator.
					// log.Warnf("unsupported is value: %s", val)
				}
			} else {
				// For other operators, append the operator and a placeholder.
				queryBuilder.WriteString(operator)
				queryBuilder.WriteString("?")
				// Replace '*' with '%' for LIKE operations.
				val = strings.ReplaceAll(val, "*", "%")
				args = append(args, val)
				index++
			}
			// Set the first flag to false after processing the first condition.
			first = false
		}
	}

	// Return the updated index, the constructed query string, and the arguments for placeholders.
	return index, queryBuilder.String(), args
}

func (q *URLQuery) Page() (page, pageSize int) {
	page = 1
	pageSize = 100
	if p, ok := q.values["page"]; ok {
		page, _ = strconv.Atoi(p[0])
	}
	if p, ok := q.values["page_size"]; ok {
		pageSize, _ = strconv.Atoi(p[0])
	}
	return page, pageSize
}

func (q *URLQuery) IsCount() bool {
	_, ok := q.values["count"]
	return ok
}

func (q *URLQuery) IsSingular() bool {
	_, ok := q.values["singular"]
	return ok
}

func (q *URLQuery) buildColumn(c string, as bool) (string, error) {
	columnName := c
	asName := ""

	// JSON path
	if strings.Contains(c, "->") {
		columnName, asName = jsonPathFunc[q.driver](c)
	}

	// function
	if strings.Contains(c, "(") {
		for _, match := range funcExp.FindAllStringSubmatch(columnName, -1) {
			funcName := strings.ToLower(match[1])
			if !allowedFunctionExp.MatchString(funcName) {
				return "", errors.New("function not allowed")
			}
			if asName == "" {
				asName = funcName
			}
		}
	}

	if as && asName != "" {
		columnName += fmt.Sprintf(" AS %s", asName)
	}
	return columnName, nil
}

func buildMysqlJSONPath(column string) (jsonPath, asName string) {
	parts := strings.Split(column, "->")
	columnName := parts[0]
	parts = parts[1:]
	for i, part := range parts {
		part = strings.Trim(strings.Trim(strings.TrimPrefix(part, ">"), `'`), `"`)
		isIndex := false
		if _, err := strconv.ParseInt(part, 10, 64); err == nil {
			isIndex = true
		}
		if isIndex {
			part = fmt.Sprintf("[%s]", part)
		} else {
			// use last non number filed as name
			asName = part
			// add dot to non number field
			part = "." + part
		}
		parts[i] = part
	}
	jsonPath = fmt.Sprintf("%s->'$%s'", columnName, strings.Join(parts, ""))
	return
}

func buildPGJSONPath(column string) (jsonPath, asName string) {
	parts := strings.Split(column, "->")
	for i, part := range parts {
		if i == 0 {
			// skip column name
			continue
		}
		doubleArrow := false
		if strings.HasPrefix(part, ">") {
			doubleArrow = true
			part = part[1:]
		}
		part = strings.Trim(strings.Trim(part, `'`), `"`)
		isIndex := false
		if _, err := strconv.ParseInt(part, 10, 64); err == nil {
			isIndex = true
		}
		if !isIndex {
			// use last non number filed as name
			asName = part
			// add quote for non number field
			part = fmt.Sprintf(`'%s'`, part)
		}
		if doubleArrow {
			part = ">" + part
		}
		parts[i] = part
	}
	jsonPath = strings.Join(parts, "->")
	return
}

func buildSqliteJSONPath(column string) (jsonPath, asName string) {
	// sqlite compatible with MySQL and PG
	return buildPGJSONPath(column)
}
