package mysql

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MySQLQueryBuilder implements SQL query building for MySQL
type MySQLQueryBuilder struct{}

// NewMySQLQueryBuilder creates a new MySQL query builder
func NewMySQLQueryBuilder() *MySQLQueryBuilder {
	return &MySQLQueryBuilder{}
}

// GetPlaceholder returns MySQL-style parameter placeholder (?)
func (b *MySQLQueryBuilder) GetPlaceholder(index int) string {
	return "?"
}

// QuoteIdentifier returns a MySQL quoted identifier
func (b *MySQLQueryBuilder) QuoteIdentifier(name string) string {
	return "`" + strings.Replace(name, "`", "``", -1) + "`"
}

// BuildQuery constructs a MySQL query from HTTP request components
func (b *MySQLQueryBuilder) BuildQuery(method string, table string, id string, relations []string, filters map[string]string, body json.RawMessage) (string, map[string]interface{}, error) {
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
			query.WriteString(" WHERE id = ?")
			params["id"] = id
		} else if len(filters) > 0 {
			query.WriteString(" WHERE ")
			keys := make([]string, 0, len(filters))
			for k := range filters {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			conditions := make([]string, 0, len(filters))
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

		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		columns := make([]string, 0, len(fields))
		placeholders := make([]string, 0, len(fields))
		for _, k := range keys {
			columns = append(columns, k)
			placeholders = append(placeholders, "?")
			params[k] = fields[k]
		}

		query.WriteString("INSERT INTO ")
		query.WriteString(table)
		query.WriteString(" (")
		query.WriteString(strings.Join(columns, ", "))
		query.WriteString(") VALUES (")
		query.WriteString(strings.Join(placeholders, ", "))
		query.WriteString("); SELECT * FROM ")
		query.WriteString(table)
		query.WriteString(" WHERE id = LAST_INSERT_ID()")

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
		for _, k := range keys {
			updates = append(updates, fmt.Sprintf("%s = ?", k))
			params[k] = fields[k]
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
