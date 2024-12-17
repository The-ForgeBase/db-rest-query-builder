package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/The-ForgeBase/restql/query"
	"github.com/The-ForgeBase/restql/utils"
)

// Function to check if a value is boolean and needs `IS` or `=`
func isBoolean(val any) bool {
	switch v := val.(type) {
	case bool:
		return true
	case *sql.NullBool:
		return v.Valid
	default:
		return false
	}
}

var (
	DBType = "surrealdb"
)

// DynamicHandler handles dynamic routes like /products, /users, etc.
func GetQL(r *http.Request, dbtype string) (*utils.ReturnQuery, error) {

	DBType = dbtype

	// Extract the table name from the URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 || parts[1] == "" {
		return nil, fmt.Errorf("table name required")
	}
	tableName := parts[1]

	// 1. Validate the table name
	if err := utils.ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("invalid table name")
	}

	switch r.Method {
	case http.MethodGet:
		q, err := getRecords(r, tableName)
		if err != nil {
			return nil, err
		}
		return q, nil
	case http.MethodPost:
		q, err := insertRecord(r, tableName)
		if err != nil {
			return nil, err
		}
		return q, nil
	case http.MethodPut:
		q, err := updateRecord(r, tableName)
		if err != nil {
			return nil, err
		}
		return q, nil
	case http.MethodDelete:
		q, err := deleteRecord(r, tableName)
		if err != nil {
			return nil, err
		}
		return q, nil
	default:
		return nil, fmt.Errorf("method not allowed")
	}
}

// Get records (supports filtering, pagination, sorting)
func getRecords(r *http.Request, tableName string) (*utils.ReturnQuery, error) {
	queryParams := r.URL.Query()

	// 1. Parse filters
	filterSQL, args := query.ParseFilters(queryParams, DBType)

	// 2. Handle pagination
	page := queryParams.Get("page")
	pageSize := queryParams.Get("page_size")

	if page == "" {
		page = "1"
	}

	if pageSize == "" {
		pageSize = "100"
	}

	limit, offset := query.ParsePagination(page, pageSize)

	// 3. Handle sorting
	orderSQL := query.ParseOrder(queryParams.Get("order"))

	if orderSQL == "" {
		orderSQL = "ORDER BY id ASC"
	}

	// 4. Build dynamic SQL query
	sql := ""

	if filterSQL != "" {
		sql = fmt.Sprintf("SELECT * FROM %s WHERE %s %s LIMIT %d OFFSET %d", tableName, filterSQL, orderSQL, limit, offset)

		if DBType == "surrealdb" {
			sql = fmt.Sprintf("SELECT * FROM %s WHERE %s %s LIMIT %d START %d", tableName, filterSQL, orderSQL, limit, offset)
		}
	} else {
		sql = fmt.Sprintf("SELECT * FROM %s %s LIMIT %d OFFSET %d", tableName, orderSQL, limit, offset)

		if DBType == "surrealdb" {
			sql = fmt.Sprintf("SELECT * FROM %s %s LIMIT %d START %d", tableName, orderSQL, limit, offset)
		}
	}

	// 5. Return the query and args
	query := utils.ReturnQuery{Query: sql, Args: args}

	return &query, nil
}

// Insert, update, and delete records with bulk support
func insertRecord(r *http.Request, tableName string) (*utils.ReturnQuery, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}

	// 1. Parse the JSON body (can be a single record or a list of records)
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		// If it fails, try to unmarshal it as a single record
		var singleRecord map[string]interface{}
		if err := json.Unmarshal(body, &singleRecord); err != nil {
			return nil, fmt.Errorf("invalid JSON format")
		}
		records = append(records, singleRecord)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records to insert")
	}

	// 2. Build column names and placeholders
	columns, placeholders, values := query.BuildInsertQueryParts(records)

	// 3. Construct the SQL query for bulk insert
	var sql string
	if len(records) == 1 {
		sql = fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", tableName, columns, placeholders[0])
	} else {
		sql = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columns, strings.Join(placeholders, ", "))
	}

	// fmt.Println(sql)

	if DBType == "surrealdb" {
		// sample insert query
		// 		INSERT INTO planet [
		// 	{
		// 		name: 'Venus',
		//         surface_temp: 462,
		//         temp_55_km_up: 27
		// 	},
		// 	{
		// 		name: 'Earth',
		//         surface_temp: 15,
		//         temp_55_km_up: -55
		// 	}
		// ]
		// TODO: improve for single record, currently default to bulk insert
		body := records // No need to append, just use records directly
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, err // Handle error appropriately
		}
		sql = fmt.Sprintf("INSERT INTO %s %s", tableName, bodyJSON)
	}

	// 4. Return the query and args
	return &utils.ReturnQuery{Query: sql, Args: values}, nil
}

func updateRecord(r *http.Request, tableName string) (*utils.ReturnQuery, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}

	// Extract the primary key from the URL path (e.g., /products/1)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		return nil, fmt.Errorf("primary key required for update")
	}
	primaryKey := parts[2]

	// 1. Parse the JSON body (can be a single update or multiple updates)
	var updates map[string]interface{}
	if err := json.Unmarshal(body, &updates); err != nil {
		return nil, fmt.Errorf("invalid JSON format")
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// 2. Build the SET clause
	setClause, values := query.BuildUpdateQueryParts(updates)

	// 3. Construct the SQL query for update
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", tableName, setClause)

	if DBType == "surrealdb" {
		// NOTE: surrealdb does not support bulk update
		body := updates // No need to append, just use records directly
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, err // Handle error appropriately
		}
		sql = fmt.Sprintf("UPDATE %s:%s MERGE %s", tableName, primaryKey, bodyJSON)
	}

	// 4. Append the primary key to the query args
	values = append(values, primaryKey)

	// 5. Return the query and args
	return &utils.ReturnQuery{Query: sql, Args: values}, nil
}

func deleteRecord(r *http.Request, tableName string) (*utils.ReturnQuery, error) {
	// Extract the primary key from the URL path (e.g., /products/1)
	parts := strings.Split(r.URL.Path, "/")

	primaryKey := ""
	if len(parts) > 2 {
		primaryKey = parts[2]
	}

	// Parse filters from query string for bulk delete
	queryParams := r.URL.Query()
	filterSQL, args := query.ParseFilters(queryParams, DBType)

	// 1. If a primary key is provided, delete only that specific record
	if primaryKey != "" {
		sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
		if DBType == "surrealdb" {
			sql = fmt.Sprintf("DELETE %s:%s", tableName, primaryKey)
		}
		return &utils.ReturnQuery{Query: sql, Args: []interface{}{primaryKey}}, nil
	}

	// 2. If query filters are present, build the WHERE clause
	if filterSQL != "" {
		sql := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, filterSQL)
		if DBType == "surrealdb" {
			sql = fmt.Sprintf("DELETE %s WHERE %s", tableName, filterSQL)
		}
		return &utils.ReturnQuery{Query: sql, Args: args}, nil
	}

	// 3. If no filters and no primary key, return an error
	return nil, fmt.Errorf("primary key or filters required for delete")
}
