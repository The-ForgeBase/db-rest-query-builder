package postgres

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// PostgresQueryBuilder implements SQL query building for PostgreSQL
type PostgresQueryBuilder struct{}

// NewPostgresQueryBuilder creates a new PostgreSQL query builder
func NewPostgresQueryBuilder() *PostgresQueryBuilder {
	return &PostgresQueryBuilder{}
}

// GetPlaceholder returns PostgreSQL-style parameter placeholder ($1, $2, etc.)
func (b *PostgresQueryBuilder) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// QuoteIdentifier returns a PostgreSQL quoted identifier
func (b *PostgresQueryBuilder) QuoteIdentifier(name string) string {
	return `"` + strings.Replace(name, `"`, `""`, -1) + `"`
}

// BuildQuery constructs a PostgreSQL query from HTTP request components
func (b *PostgresQueryBuilder) BuildQuery(method string, table string, id string, relations []string, filters map[string]string, body json.RawMessage) (string, map[string]interface{}, error) {
	var query strings.Builder
	params := make(map[string]interface{})

	switch method {
	case "GET":
		query.WriteString("SELECT ")
		if len(relations) > 0 {
			sort.Strings(relations)
			query.WriteString(strings.Join(relations, ", "))
		} else {
			query.WriteString("*")
		}
		query.WriteString(" FROM ")
		query.WriteString(table)

		if id != "" {
			query.WriteString(" WHERE id = $1")
			params["id"] = id
		} else if len(filters) > 0 {
			query.WriteString(" WHERE ")
			keys := make([]string, 0, len(filters))
			for k := range filters {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			conditions := make([]string, 0, len(filters))
			paramCount := 1
			for _, k := range keys {
				conditions = append(conditions, fmt.Sprintf("%s = $%d", k, paramCount))
				params[k] = filters[k]
				paramCount++
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

		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		columns := make([]string, 0, len(fields))
		placeholders := make([]string, 0, len(fields))
		paramCount := 1
		for _, k := range keys {
			columns = append(columns, k)
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramCount))
			params[k] = fields[k]
			paramCount++
		}

		query.WriteString("INSERT INTO ")
		query.WriteString(table)
		query.WriteString(" (")
		query.WriteString(strings.Join(columns, ", "))
		query.WriteString(") VALUES (")
		query.WriteString(strings.Join(placeholders, ", "))
		query.WriteString(") RETURNING *")

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

		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		updates := make([]string, 0, len(fields))
		paramCount := 1
		for _, k := range keys {
			updates = append(updates, fmt.Sprintf("%s = $%d", k, paramCount))
			params[k] = fields[k]
			paramCount++
		}
		params["id"] = id

		query.WriteString("UPDATE ")
		query.WriteString(table)
		query.WriteString(" SET ")
		query.WriteString(strings.Join(updates, ", "))
		query.WriteString(" WHERE id = $")
		query.WriteString(fmt.Sprintf("%d", paramCount))
		query.WriteString(" RETURNING *")

	case "DELETE":
		if id == "" {
			return "", nil, fmt.Errorf("id is required for DELETE")
		}

		query.WriteString("DELETE FROM ")
		query.WriteString(table)
		query.WriteString(" WHERE id = $1")
		params["id"] = id

	default:
		return "", nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	return query.String(), params, nil
}
