package query

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/The-ForgeBase/restql/pkg/restql/utils"
)

// Default values
const (
	DefaultPage     = 1
	DefaultPageSize = 100
	MaxPageSize     = 1000 // To prevent excessive load on DB
)

// ParseFilters converts query parameters into SQL WHERE clause
func ParseFilters(queryParams url.Values, dbType string) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}

	// Iterate over each query parameter
	for key, values := range queryParams {
		for _, value := range values {
			if key == "and" || key == "or" || key == "not" {
				// Handle nested groups like and=(...), or=(...), not=(...)
				groupSQL, groupArgs := parseGroup(key, value, dbType)
				clauses = append(clauses, fmt.Sprintf("(%s)", groupSQL))
				args = append(args, groupArgs...)
			} else {
				// Handle standard column filters (e.g., level=lt.2)
				clause, clauseArgs := parseCondition(key, value, dbType)
				if clause != "" {
					clauses = append(clauses, clause)
					args = append(args, clauseArgs...)
				}
			}
		}
	}

	return strings.Join(clauses, " AND "), args
}

// Parse a group (like and=(level=lt.2,or=(hidden=is.false)))
func parseGroup(logic string, value string, dbType string) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}

	// Remove parentheses from the value, e.g., "level=lt.2,or=(hidden=is.false)"
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")

	// Split into parts (comma-separated)
	parts := splitPreservingGroups(value)

	for _, part := range parts {
		if strings.HasPrefix(part, "and=") || strings.HasPrefix(part, "or=") || strings.HasPrefix(part, "not=") {
			// Handle nested logic groups
			key := part[:3] // "and", "or", or "not"
			subValue := strings.TrimPrefix(part, key+"=")
			subSQL, subArgs := parseGroup(key, subValue, dbType)
			clauses = append(clauses, fmt.Sprintf("(%s)", subSQL))
			args = append(args, subArgs...)
		} else {
			// Handle basic conditions (like level=lt.2)
			clause, clauseArgs := parseConditionFromPart(part, dbType)
			if clause != "" {
				clauses = append(clauses, clause)
				args = append(args, clauseArgs...)
			}
		}
	}

	return strings.Join(clauses, fmt.Sprintf(" %s ", strings.ToUpper(logic))), args
}

// Parse a condition like "level=lt.2"
func parseCondition(key string, value string, dbType string) (string, []interface{}) {
	return parseConditionFromPart(fmt.Sprintf("%s=%s", key, value), dbType)
}

func parseConditionFromPart(part string, dbType string) (string, []interface{}) {
	r := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)=([a-z]+)\.(.+)$`)
	matches := r.FindStringSubmatch(part)
	if len(matches) != 4 {
		return "", nil
	}

	column := matches[1]
	operator := matches[2]
	rawValue := matches[3]

	sqlOperator, ok := utils.Operators[operator]
	if !ok {
		return "", nil
	}

	// Handle LIKE operator
	if operator == "like" {
		rawValue = strings.ReplaceAll(rawValue, "*", "%")
	}

	// Handle IS operator for booleans
	if operator == "is" {
		if rawValue == "true" || rawValue == "false" {
			rawValue = strings.ToUpper(rawValue)
		}
	}

	// Handle type conversion based on column type
	// convertedValue := convertTypeForColumn(dbType, column, rawValue)
	convertedValue, err := utils.ParseQueryParam(rawValue)

	if err != nil {
		panic(err)
	}

	// TODO: handle IS operator based on database type
	if sqlOperator == "IS" || sqlOperator == "LIKE" {
		sqlOperator = "="
	}

	// fmt.Printf("Column: %s, Operator: %s, Raw Value: %s, Converted Value: %v\n", column, operator, rawValue, convertedValue)

	return fmt.Sprintf("%s %s ?", column, sqlOperator), []interface{}{convertedValue}
}

// Convert value based on the column's data type
func convertTypeForColumn(dbType, column, rawValue string) any {
	fmt.Printf("Column: %s, Raw Value: %s\n", column, rawValue)
	// Lookup the column type in the DB schema
	columnType := getColumnType(dbType, column)
	converter, exists := utils.TypeConverters[columnType]
	if exists {

		// Check for specific type conversion
		if columnType == "INTEGER" {
			if intValue, err := strconv.ParseInt(rawValue, 10, 64); err == nil {
				return intValue
			}
		}
		// Convert the value using the appropriate type converter
		return converter(rawValue)
	}

	// Default case: return the raw value (could be enhanced based on your needs)
	return rawValue
}

// Get the column type based on the database type and column name
func getColumnType(dbType, column string) string {
	// For simplicity, assuming a default column type map
	// This should be enhanced based on the actual DB schema
	return "INTEGER" // Just an example, use actual DB schema here
}

// Split on `,` but respect nested groups, e.g., a=lt.2,or=(b=is.false)
func splitPreservingGroups(input string) []string {
	parts := []string{}
	groupLevel := 0
	current := ""

	for _, char := range input {
		switch char {
		case '(':
			groupLevel++
			current += string(char)
		case ')':
			groupLevel--
			current += string(char)
		case ',':
			if groupLevel == 0 {
				parts = append(parts, current)
				current = ""
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// ParseOrder parses ?order=id.desc,name.asc into SQL ORDER BY clause
func ParseOrder(order string) string {
	if order == "" {
		return ""
	}

	parts := strings.Split(order, ",")
	var orderClauses []string
	for _, part := range parts {
		subParts := strings.SplitN(part, ".", 2)
		column := subParts[0]
		direction := "ASC"
		if len(subParts) == 2 && subParts[1] == "desc" {
			direction = "DESC"
		}
		orderClauses = append(orderClauses, fmt.Sprintf("%s %s", column, direction))
	}

	return fmt.Sprintf("ORDER BY %s", strings.Join(orderClauses, ", "))
}

// ParsePagination converts ?page=2&page_size=10 into SQL LIMIT and OFFSET
func ParsePagination(pageStr, pageSizeStr string) (limit, offset int) {
	// 1️⃣ Parse `page` and `page_size` with defaults
	page := DefaultPage
	pageSize := DefaultPageSize

	// 2️⃣ Convert `page` to int, fallback to default if parsing fails
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	// 3️⃣ Convert `page_size` to int, fallback to default if parsing fails
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
		pageSize = ps
	}

	// 4️⃣ Enforce a maximum page size to avoid large requests
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// 5️⃣ Calculate LIMIT and OFFSET
	limit = pageSize
	offset = (page - 1) * pageSize

	return limit, offset
}

func BuildInsertQueryParts(records []map[string]interface{}) (string, []string, []interface{}) {
	if len(records) == 0 {
		return "", nil, nil
	}

	columns := []string{}
	for column := range records[0] {
		columns = append(columns, column)
	}

	placeholders := []string{}
	values := []interface{}{}

	for _, record := range records {
		rowPlaceholders := []string{}
		for _, col := range columns {
			rowPlaceholders = append(rowPlaceholders, "?")
			values = append(values, record[col])
		}
		placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", ")))
	}

	return strings.Join(columns, ", "), placeholders, values
}

func BuildUpdateQueryParts(updates map[string]interface{}) (string, []interface{}) {
	if len(updates) == 0 {
		return "", nil
	}

	setClauses := []string{}
	values := []interface{}{}

	for column, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", column))
		values = append(values, value)
	}

	return strings.Join(setClauses, ", "), values
}
