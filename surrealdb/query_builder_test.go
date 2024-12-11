package surrealdb

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestQueryBuilder_BuildQuery(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		table      string
		id         string
		relations  []string
		filters    map[string]string
		body       json.RawMessage
		wantQuery  string
		wantParams map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "GET all records",
			method:     "GET",
			table:      "users",
			wantQuery:  "SELECT * FROM users",
			wantParams: map[string]interface{}{},
		},
		{
			name:       "GET single record",
			method:     "GET",
			table:      "users",
			id:         "123",
			wantQuery:  "SELECT * FROM users:123",
			wantParams: map[string]interface{}{},
		},
		{
			name:       "GET with relations",
			method:     "GET",
			table:      "users",
			relations:  []string{"name", "email", "profile.*"},
			wantQuery:  "SELECT name, email, profile.* FROM users",
			wantParams: map[string]interface{}{},
		},
		{
			name:   "GET with filters",
			method: "GET",
			table:  "users",
			filters: map[string]string{
				"age":    "25",
				"active": "true",
			},
			wantQuery: "SELECT * FROM users WHERE age = $p1 AND active = $p2",
			wantParams: map[string]interface{}{
				"p1": "25",
				"p2": "true",
			},
		},
		{
			name:      "POST new record",
			method:    "POST",
			table:     "users",
			body:      json.RawMessage(`{"name":"John Doe","email":"john@example.com"}`),
			wantQuery: "CREATE users SET name = $p1, email = $p2 RETURN *",
			wantParams: map[string]interface{}{
				"p1": "John Doe",
				"p2": "john@example.com",
			},
		},
		{
			name:    "POST without body",
			method:  "POST",
			table:   "users",
			wantErr: true,
		},
		{
			name:      "PUT update record",
			method:    "PUT",
			table:     "users",
			id:        "123",
			body:      json.RawMessage(`{"name":"John Smith","email":"john.smith@example.com"}`),
			wantQuery: "UPDATE users:123 SET name = $p1, email = $p2 RETURN *",
			wantParams: map[string]interface{}{
				"p1": "John Smith",
				"p2": "john.smith@example.com",
			},
		},
		{
			name:    "PUT without ID",
			method:  "PUT",
			table:   "users",
			body:    json.RawMessage(`{"name":"John Smith"}`),
			wantErr: true,
		},
		{
			name:      "PATCH partial update",
			method:    "PATCH",
			table:     "users",
			id:        "123",
			body:      json.RawMessage(`{"email":"new.email@example.com"}`),
			wantQuery: "UPDATE users:123 MERGE email = $p1 RETURN *",
			wantParams: map[string]interface{}{
				"p1": "new.email@example.com",
			},
		},
		{
			name:    "PATCH without ID",
			method:  "PATCH",
			table:   "users",
			body:    json.RawMessage(`{"email":"new.email@example.com"}`),
			wantErr: true,
		},
		{
			name:       "DELETE record",
			method:     "DELETE",
			table:      "users",
			id:         "123",
			wantQuery:  "DELETE users:123 RETURN *",
			wantParams: map[string]interface{}{},
		},
		{
			name:    "DELETE without ID",
			method:  "DELETE",
			table:   "users",
			wantErr: true,
		},
		{
			name:    "Unsupported method",
			method:  "INVALID",
			table:   "users",
			wantErr: true,
		},
		{
			name:    "POST with invalid JSON",
			method:  "POST",
			table:   "users",
			body:    json.RawMessage(`{"invalid json"`),
			wantErr: true,
		},
		{
			name:   "GET with special characters in filters",
			method: "GET",
			table:  "users",
			filters: map[string]string{
				"name": "O'Connor",
				"type": "user@example.com",
			},
			wantQuery: "SELECT * FROM users WHERE name = $p1 AND type = $p2",
			wantParams: map[string]interface{}{
				"p1": "O'Connor",
				"p2": "user@example.com",
			},
		},
	}

	qb := NewSurrealQlQueryBuilder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotParams, err := qb.BuildQuery(tt.method, tt.table, tt.id, tt.relations, tt.filters, tt.body)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if gotQuery != tt.wantQuery {
				t.Errorf("BuildQuery() gotQuery = %v, want %v", gotQuery, tt.wantQuery)
			}

			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("BuildQuery() gotParams = %v, want %v", gotParams, tt.wantParams)
			}

			// fmt.Printf("Query: %s\n", gotQuery)
			// fmt.Printf("Params: %v\n", gotParams)
		})
	}
}
