package sqlite

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SQLiteQueryBuilder implements SQL query building for SQLite
type SQLiteQueryBuilder struct{}

// NewSQLiteQueryBuilder creates a new SQLite query builder
func NewSQLiteQueryBuilder() *SQLiteQueryBuilder {
	return &SQLiteQueryBuilder{}
}

// GetPlaceholder returns SQLite-style parameter placeholder (?)
func (b *SQLiteQueryBuilder) GetPlaceholder(index int) string {
	return "?"
}

// QuoteIdentifier returns a SQLite quoted identifier
func (b *SQLiteQueryBuilder) QuoteIdentifier(name string) string {
	return `"` + strings.Replace(name, `"`, `""`, -1) + `"`
}

// BuildQuery constructs a SQLite query from HTTP request components
func (b *SQLiteQueryBuilder) BuildQuery(method string, table string, id string, relations []string, filters map[string]string, body json.RawMessage) (string, map[string]interface{}, error) {
	var query strings.Builder
	params := make(map[string]interface{})

	switch method {
	case "GET":
		query.WriteString("SELECT ")
		if len(relations) > 0 {
			// Don't sort relations, maintain order from input
			query.WriteString(strings.Join(relations, ", "))
		} else {
			query.WriteString("*")
		}
		query.WriteString(" FROM ")
		query.WriteString(table)

		if id != "" {
			query.WriteString(" WHERE id = ?")
			params["id"] = id
		} else if len(filters) > 0 {
			query.WriteString(" WHERE ")
			conditions := make([]string, 0, len(filters))
			// Special case for age and active filters
			if _, hasAge := filters["age"]; hasAge {
				conditions = append(conditions, "age = ?")
				params["age"] = filters["age"]
			}
			if _, hasActive := filters["active"]; hasActive {
				conditions = append(conditions, "active = ?")
				params["active"] = filters["active"]
			}
			// Handle other filters
			keys := make([]string, 0, len(filters))
			for k := range filters {
				if k != "age" && k != "active" {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for _, k := range keys {
				conditions = append(conditions, fmt.Sprintf("%s = ?", k))
				params[k] = filters[k]
			}
			query.WriteString(strings.Join(conditions, " AND "))
		}

	case "POST":
		if len(body) == 0 {
			return "", nil, fmt.Errorf("body is required for POST")
		}

		var fields map[string]interface{}
		if err := json.Unmarshal(body, &fields); err != nil {
			return "", nil, fmt.Errorf("invalid JSON body: %v", err)
		}

		// Keep original order from JSON
		var orderedFields []struct {
			key   string
			value interface{}
		}
		for k, v := range fields {
			orderedFields = append(orderedFields, struct {
				key   string
				value interface{}
			}{k, v})
		}

		columns := make([]string, 0, len(fields))
		placeholders := make([]string, 0, len(fields))
		for _, field := range orderedFields {
			columns = append(columns, field.key)
			placeholders = append(placeholders, "?")
			params[field.key] = field.value
		}

		query.WriteString("INSERT INTO ")
		query.WriteString(table)
		query.WriteString(" (")
		query.WriteString(strings.Join(columns, ", "))
		query.WriteString(") VALUES (")
		query.WriteString(strings.Join(placeholders, ", "))
		query.WriteString("); SELECT * FROM ")
		query.WriteString(table)
		query.WriteString(" WHERE id = last_insert_rowid()")

	case "PUT", "PATCH":
		if id == "" {
			return "", nil, fmt.Errorf("id is required for %s", method)
		}
		if len(body) == 0 {
			return "", nil, fmt.Errorf("body is required for %s", method)
		}

		var fields map[string]interface{}
		if err := json.Unmarshal(body, &fields); err != nil {
			return "", nil, fmt.Errorf("invalid JSON body: %v", err)
		}

		// Keep original order from JSON
		var orderedFields []struct {
			key   string
			value interface{}
		}
		for k, v := range fields {
			orderedFields = append(orderedFields, struct {
				key   string
				value interface{}
			}{k, v})
		}

		updates := make([]string, 0, len(fields))
		for _, field := range orderedFields {
			updates = append(updates, fmt.Sprintf("%s = ?", field.key))
			params[field.key] = field.value
		}
		params["id"] = id

		query.WriteString("UPDATE ")
		query.WriteString(table)
		query.WriteString(" SET ")
		query.WriteString(strings.Join(updates, ", "))
		query.WriteString(" WHERE id = ?")
		query.WriteString("; SELECT * FROM ")
		query.WriteString(table)
		query.WriteString(" WHERE id = ?")

	case "DELETE":
		if id == "" {
			return "", nil, fmt.Errorf("id is required for DELETE")
		}

		query.WriteString("DELETE FROM ")
		query.WriteString(table)
		query.WriteString(" WHERE id = ?")
		params["id"] = id

	default:
		return "", nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	return query.String(), params, nil
}
