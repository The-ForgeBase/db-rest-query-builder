package surrealdb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QueryBuilder implements query building for SurrealDB
type QueryBuilder struct{}

// NewQueryBuilder creates a new SurrealDB query builder
func NewSurrealQlQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// BuildQuery constructs a SurrealQL query from HTTP request components
func (qb *QueryBuilder) BuildQuery(method string, table string, id string, relations []string, filters map[string]string, body json.RawMessage) (string, map[string]interface{}, error) {
	var query strings.Builder
	params := make(map[string]interface{})
	paramIndex := 1

	switch method {
	case "GET":
		query.WriteString("SELECT ")
		if len(relations) > 0 {
			query.WriteString(strings.Join(relations, ", "))
		} else {
			query.WriteString("*")
		}
		query.WriteString(" FROM ")
		query.WriteString(table)
		if id != "" {
			query.WriteString(":")
			query.WriteString(id)
		} else if len(filters) > 0 {
			query.WriteString(" WHERE ")
			conditions := make([]string, 0, len(filters))
			for key, value := range filters {
				paramName := fmt.Sprintf("p%d", paramIndex)
				conditions = append(conditions, fmt.Sprintf("%s = $%s", key, paramName))
				params[paramName] = value
				paramIndex++
			}
			query.WriteString(strings.Join(conditions, " AND "))
		}

	case "POST":
		if len(body) == 0 {
			return "", nil, fmt.Errorf("POST request requires a body")
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", nil, fmt.Errorf("invalid JSON body: %w", err)
		}

		query.WriteString("CREATE ")
		query.WriteString(table)
		if len(data) > 0 {
			query.WriteString(" SET ")
			fields := make([]string, 0, len(data))
			for key, value := range data {
				paramName := fmt.Sprintf("p%d", paramIndex)
				fields = append(fields, fmt.Sprintf("%s = $%s", key, paramName))
				params[paramName] = value
				paramIndex++
			}
			query.WriteString(strings.Join(fields, ", "))
		}
		query.WriteString(" RETURN *")

	case "PUT":
		if id == "" {
			return "", nil, fmt.Errorf("PUT request requires an ID")
		}
		if len(body) == 0 {
			return "", nil, fmt.Errorf("PUT request requires a body")
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", nil, fmt.Errorf("invalid JSON body: %w", err)
		}

		query.WriteString("UPDATE ")
		query.WriteString(table)
		query.WriteString(":")
		query.WriteString(id)
		if len(data) > 0 {
			query.WriteString(" SET ")
			fields := make([]string, 0, len(data))
			for key, value := range data {
				paramName := fmt.Sprintf("p%d", paramIndex)
				fields = append(fields, fmt.Sprintf("%s = $%s", key, paramName))
				params[paramName] = value
				paramIndex++
			}
			query.WriteString(strings.Join(fields, ", "))
		}
		query.WriteString(" RETURN *")

	case "PATCH":
		if id == "" {
			return "", nil, fmt.Errorf("PATCH request requires an ID")
		}
		if len(body) == 0 {
			return "", nil, fmt.Errorf("PATCH request requires a body")
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", nil, fmt.Errorf("invalid JSON body: %w", err)
		}

		query.WriteString("UPDATE ")
		query.WriteString(table)
		query.WriteString(":")
		query.WriteString(id)
		query.WriteString(" MERGE ")
		if len(data) > 0 {
			fields := make([]string, 0, len(data))
			for key, value := range data {
				paramName := fmt.Sprintf("p%d", paramIndex)
				fields = append(fields, fmt.Sprintf("%s = $%s", key, paramName))
				params[paramName] = value
				paramIndex++
			}
			query.WriteString(strings.Join(fields, ", "))
		}
		query.WriteString(" RETURN *")

	case "DELETE":
		if id == "" {
			return "", nil, fmt.Errorf("DELETE request requires an ID")
		}
		query.WriteString("DELETE ")
		query.WriteString(table)
		query.WriteString(":")
		query.WriteString(id)
		query.WriteString(" RETURN *")

	default:
		return "", nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	return query.String(), params, nil
}
