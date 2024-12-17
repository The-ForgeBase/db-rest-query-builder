package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test GetQL function (all methods)
func TestGetQL(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       io.Reader
		wantErr    bool
		errMessage string
	}{
		{"missing table name", http.MethodGet, "/", nil, true, "table name required"},
		{"invalid table name", http.MethodGet, "/123invalidTable", nil, true, "invalid table name"},
		{"method not allowed", http.MethodPatch, "/products", nil, true, "method not allowed"},
		{"valid GET request", http.MethodGet, "/products", nil, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			_, err := GetQL(req, "surrealdb")
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test getRecords function with filters and pagination
func TestGetRecords(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			"simple eq filter",
			"/products?level=eq.2",
			"SELECT * FROM products WHERE level = ? ORDER BY id ASC LIMIT 100 START 0",
			[]interface{}{int64(2)},
		},
		{
			"multiple filters with AND",
			"/products?level=lt.2&hidden=is.false",
			"SELECT * FROM products WHERE level < ? AND hidden = ? ORDER BY id ASC LIMIT 100 START 0",
			[]interface{}{int64(2), false},
		},
		{
			"OR condition",
			"/products?or=(level=lt.2,hidden=is.false)",
			"SELECT * FROM products WHERE (level < ? OR hidden = ?) ORDER BY id ASC LIMIT 100 START 0",
			[]interface{}{int64(2), false},
		},
		{
			"pagination and sorting",
			"/products?page=2&page_size=10&order=level.asc",
			"SELECT * FROM products ORDER BY level ASC LIMIT 10 START 10",
			[]interface{}{},
		},
		{
			"filter with sorting",
			"/products?level=gt.5&order=price.desc",
			"SELECT * FROM products WHERE level > ? ORDER BY price DESC LIMIT 100 START 0",
			[]interface{}{int64(5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			query, err := getRecords(req, "products")
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, query.Query)
			assert.Equal(t, tt.expectedArgs, query.Args)
		})
	}
}

// Test insertRecord function (with bulk support)
func TestInsertRecord(t *testing.T) {
	tests := []struct {
		name         string
		body         interface{}
		wantErr      bool
		errMessage   string
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			"single record insertion",
			map[string]interface{}{"name": "Product1", "price": float64(100)},
			false,
			"",
			"INSERT INTO products [{\"name\":\"Product1\",\"price\":100}]",
			[]interface{}{"Product1", float64(100)},
		},
		{
			"bulk insertion",
			[]map[string]interface{}{
				{"name": "Product1", "price": float64(100)},
				{"name": "Product2", "price": float64(200)},
			},
			false,
			"",
			"INSERT INTO products [{\"name\":\"Product1\",\"price\":100},{\"name\":\"Product2\",\"price\":200}]",
			[]interface{}{"Product1", float64(100), "Product2", float64(200)},
		},
		{
			"invalid JSON",
			"invalid-json",
			true,
			"invalid JSON format",
			"",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
			query, err := insertRecord(req, "products")

			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSQL, query.Query)
				assert.Equal(t, tt.expectedArgs, query.Args)
			}
		})
	}
}

// Test updateRecord function (with filtering and primary key)
func TestUpdateRecord(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		body         interface{}
		expectedSQL  string
		expectedArgs []interface{}
		wantErr      bool
		errMessage   string
	}{
		{
			"update by primary key",
			"/products/1",
			map[string]interface{}{"name": "Updated Product", "price": float64(150)},
			"UPDATE products:1 MERGE {\"name\":\"Updated Product\",\"price\":150}",
			[]interface{}{"Updated Product", float64(150), "1"},
			false,
			"",
		},
		// {
		// 	"bulk update by primary key",
		// 	"/products/1",
		// 	map[string]interface{}{"name": "Updated Product", "price": float64(150)},
		// 	"UPDATE products SET name = ?, price = ? WHERE id = ?",
		// 	[]interface{}{"Updated Product", float64(150), "1"},
		// 	false,
		// 	"",
		// },
		{
			"no fields to update",
			"/products/1",
			map[string]interface{}{},
			"",
			nil,
			true,
			"no fields to update",
		},
		{
			"invalid JSON",
			"/products/1",
			"invalid-json",
			"",
			nil,
			true,
			"invalid JSON format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, tt.path, bytes.NewReader(body))
			query, err := updateRecord(req, "products")

			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errMessage)
			} else {
				assert.NoError(t, err)
				// fmt.Println(query.Query)
				// fmt.Println(tt.expectedSQL)
				assert.Equal(t, tt.expectedSQL, query.Query)
				assert.Equal(t, tt.expectedArgs, query.Args)
			}
		})
	}
}

// Test deleteRecord function (with filters and primary key)
func TestDeleteRecord(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		query        string
		expectedSQL  string
		expectedArgs []interface{}
		wantErr      bool
		errMessage   string
	}{
		{
			"delete by primary key",
			"/products/1",
			"",
			"DELETE products:1",
			[]interface{}{"1"},
			false,
			"",
		},
		{
			"delete by filter",
			"/products",
			"level=lt.5",
			"DELETE products WHERE level < ?",
			[]interface{}{int64(5)},
			false,
			"",
		},
		{
			"delete with no primary key or filters",
			"/products",
			"",
			"",
			nil,
			true,
			"primary key or filters required for delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tt.path+"?"+tt.query, nil)
			query, err := deleteRecord(req, "products")

			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSQL, query.Query)
				assert.Equal(t, tt.expectedArgs, query.Args)
			}
		})
	}
}
