package postgres

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPostgresQueryBuilder_BuildQuery(t *testing.T) {
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
			wantQuery:  "SELECT * FROM users WHERE id = $1",
			wantParams: map[string]interface{}{"id": "123"},
		},
		{
			name:      "GET with relations",
			method:    "GET",
			table:     "users",
			relations: []string{"email", "name", "profile"},
			wantQuery: "SELECT email, name, profile FROM users",
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
			wantQuery: "SELECT * FROM users WHERE active = $1 AND age = $2",
			wantParams: map[string]interface{}{
				"age":    "25",
				"active": "true",
			},
		},
		{
			name:   "GET with filters and relations",
			method: "GET",
			table:  "users",
			relations: []string{"email", "name"},
			filters: map[string]string{
				"age": "25",
			},
			wantQuery: "SELECT email, name FROM users WHERE age = $1",
			wantParams: map[string]interface{}{
				"age": "25",
			},
		},
		{
			name:   "POST new record",
			method: "POST",
			table:  "users",
			body:   json.RawMessage(`{"email":"john@example.com","name":"John Doe"}`),
			wantQuery: "INSERT INTO users (email, name) VALUES ($1, $2) RETURNING *",
			wantParams: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
		{
			name:    "POST without body",
			method:  "POST",
			table:   "users",
			wantErr: true,
		},
		{
			name:   "PUT update record",
			method: "PUT",
			table:  "users",
			id:     "123",
			body:   json.RawMessage(`{"email":"john.smith@example.com","name":"John Smith"}`),
			wantQuery: "UPDATE users SET email = $1, name = $2 WHERE id = $3 RETURNING *",
			wantParams: map[string]interface{}{
				"name":  "John Smith",
				"email": "john.smith@example.com",
				"id":    "123",
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
			name:   "PATCH partial update",
			method: "PATCH",
			table:  "users",
			id:     "123",
			body:   json.RawMessage(`{"email":"new.email@example.com"}`),
			wantQuery: "UPDATE users SET email = $1 WHERE id = $2 RETURNING *",
			wantParams: map[string]interface{}{
				"email": "new.email@example.com",
				"id":    "123",
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
			name:      "DELETE record",
			method:    "DELETE",
			table:     "users",
			id:        "123",
			wantQuery: "DELETE FROM users WHERE id = $1",
			wantParams: map[string]interface{}{
				"id": "123",
			},
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
			wantQuery: "SELECT * FROM users WHERE name = $1 AND type = $2",
			wantParams: map[string]interface{}{
				"name": "O'Connor",
				"type": "user@example.com",
			},
		},
	}

	qb := NewPostgresQueryBuilder()

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
		})
	}
}
