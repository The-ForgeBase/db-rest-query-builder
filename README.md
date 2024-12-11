# Database Query Builders and HTTP Adapters

This package provides a collection of query builders and HTTP adapters for various databases (SurrealDB, PostgreSQL, MySQL, and SQLite). It allows you to convert RESTful HTTP requests into native database queries.

## Table of Contents

- [Installation](#installation)
- [Common Patterns](#common-patterns)
- [SurrealDB](#surrealdb)
- [PostgreSQL](#postgresql)
- [MySQL](#mysql)
- [SQLite](#sqlite)
- [Database Adapters](#database-adapters)
- [Supported Databases](#supported-databases)
- [Query Building Pattern](#query-building-pattern)
- [Database-Specific Features](#database-specific-features)
- [Common Query Patterns](#common-query-patterns)
- [Security Considerations](#security-considerations)
- [Testing](#testing)

## Installation

```bash
go get github.com/yourusername/go-bass/internal/adapters
```

## Common Patterns

All adapters follow these RESTful patterns:

### Path Parameters

- Table selection: `/{table}`
- Record identification: `/{table}/{id}`
- Filtering: `/{table}/field=value`
- Relations: `/{table}/{id}/relation`

### HTTP Methods and Body

- GET: Fetch records (no body)

  ```
  GET /users
  GET /users/123
  GET /users/status=active
  ```

- POST: Create record (JSON body)

  ```
  POST /users
  Content-Type: application/json

  {
    "name": "John Doe",
    "age": 30,
    "email": "john@example.com"
  }
  ```

- PUT: Update record (JSON body)

  ```
  PUT /users/123
  Content-Type: application/json

  {
    "age": 31,
    "status": "premium"
  }
  ```

- DELETE: Remove record (no body)
  ```
  DELETE /users/123
  ```

### Response Format

All responses are in JSON format:

```json
{
  "status": "success",
  "data": [...],
  "error": null
}
```

## SurrealDB

```go
// Initialize SurrealDB adapter
surrealdbBuilder := surrealdb.NewSurrealQlQueryBuilder()

// Example handler
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse path parameters
    vars := mux.Vars(r)
    table := vars["table"]
    id := vars["id"]

    // Read body for mutations
    var body json.RawMessage
    if r.Method == http.MethodPost || r.Method == http.MethodPut {
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
    }

    // Build query
    query, params, err := surrealdbBuilder.BuildQuery(
        r.Method,
        table,
        id,
        []string{}, // relations
        map[string]string{}, // filters
        body,
    )

    // Execute query using your database connection
    rows, err := db.Query(query, params...)
    // Handle response...
}
```

## PostgreSQL

```go
// Initialize PostgreSQL adapter
pgBuilder := postgres.NewPostgresQueryBuilder()

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse path parameters
    vars := mux.Vars(r)
    table := vars["table"]
    id := vars["id"]

    // Read body for mutations
    var body json.RawMessage
    if r.Method == http.MethodPost || r.Method == http.MethodPut {
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
    }

    // Build query
    query, params, err := pgBuilder.BuildQuery(
        r.Method,
        table,
        id,
        []string{}, // relations
        map[string]string{}, // filters
        body,
    )

    // Execute query using your database connection
    rows, err := db.Query(query, params...)
    // Handle response...
}
```

## MySQL

```go
// Initialize MySQL adapter
mysqlBuilder := mysql.NewMySQLQueryBuilder()

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse path
    path := strings.Trim(r.URL.Path, "/")
    components := strings.Split(path, "/")

    table := components[0]
    var id string
    if len(components) > 1 {
        id = components[1]
    }

    // Read body for mutations
    var body json.RawMessage
    if r.Method == http.MethodPost || r.Method == http.MethodPut {
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
    }

    // Build query
    query, params, err := mysqlBuilder.BuildQuery(
        r.Method,
        table,
        id,
        []string{}, // relations
        map[string]string{}, // filters
        body,
    )

    // Execute query
    result, err := db.Exec(query, params...)
    // Handle response...
}
```

## SQLite

````go
// Initialize SQLite adapter
sqliteBuilder := sqlite.NewSQLiteQueryBuilder()

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse request
    vars := mux.Vars(r)
    table := vars["table"]
    id := vars["id"]

    // Read body for mutations
    var body json.RawMessage
    if r.Method == http.MethodPost || r.Method == http.MethodPut {
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
    }

    // Build query
    query, params, err := sqliteBuilder.BuildQuery(
        r.Method,
        table,
        id,
        []string{}, // relations
        map[string]string{}, // filters
        body,
    )

    // For SQLite, use transactions for multi-statement queries
    tx, err := db.Begin()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    // Execute query
    result, err := tx.Exec(query, params...)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    tx.Commit()
    // Handle response...
}

## Database Adapters

This directory contains database adapters for various databases supported by Bass. Each adapter implements a consistent interface for building and executing queries based on HTTP requests.

## Query Builder Interface

All query builders implement the following interface:

```go
BuildQuery(
    method string,      // HTTP method (GET, POST, PUT, PATCH, DELETE)
    table string,       // Table/collection name
    id string,          // Resource ID (optional)
    relations []string, // Fields/relations to select (optional)
    filters map[string]string, // Query filters (optional)
    body json.RawMessage,     // Request body for mutations (optional)
) (string, map[string]interface{}, error)
````

## Supported Databases

### PostgreSQL

- Uses `$1`, `$2`, etc. for parameter placeholders
- Supports `RETURNING` clause for mutations
- Native JSON type support
- Full-text search capabilities

### MySQL

- Uses `?` for parameter placeholders
- Executes separate query for retrieving inserted/updated data
- JSON functions through JSON\_\* operators
- Full-text search through MATCH/AGAINST

### SQLite

- Uses `?` for parameter placeholders
- Simulates `RETURNING` with a subsequent SELECT
- JSON support through JSON functions
- Full-text search through FTS modules

### SurrealDB

- Uses `$p1`, `$p2`, etc. for named parameters
- Native graph database capabilities
- Built-in `RETURN` clause for all mutations
- `MERGE` operation for partial updates (PATCH)
- Table:ID syntax for record identification
- Native JSON support

## Query Building Pattern

All adapters follow a consistent pattern for building queries from HTTP requests:

### Resource Identification

- Table name and ID are provided as separate parameters
- Relations parameter specifies fields to select
- Filters map provides WHERE clause conditions

### HTTP Method Mapping

#### GET

```http
# Single record
GET /users/123
→ SELECT * FROM users WHERE id = $1

# With relations
GET /users/123?fields=name,email
→ SELECT name, email FROM users WHERE id = $1

# With filters
GET /users?age=25&active=true
→ SELECT * FROM users WHERE age = $1 AND active = $2
```

#### POST

```http
POST /users
Content-Type: application/json
{
  "name": "John Doe",
  "email": "john@example.com"
}
→ INSERT INTO users (name, email) VALUES ($1, $2)
```

#### PUT

```http
PUT /users/123
Content-Type: application/json
{
  "name": "John Smith",
  "email": "john.smith@example.com"
}
→ UPDATE users SET name = $1, email = $2 WHERE id = $3
```

#### PATCH

```http
PATCH /users/123
Content-Type: application/json
{
  "email": "new.email@example.com"
}
→ UPDATE users SET email = $1 WHERE id = $2
```

#### DELETE

```http
DELETE /users/123
→ DELETE FROM users WHERE id = $1
```

## Database-Specific Query Examples

### PostgreSQL

```sql
-- GET with relations
SELECT name, email FROM users WHERE id = $1;

-- POST with RETURNING
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- PUT with RETURNING
UPDATE users SET name = $1, email = $2 WHERE id = $3 RETURNING *;
```

### MySQL

```sql
-- GET with relations
SELECT name, email FROM users WHERE id = ?;

-- POST with separate select
INSERT INTO users (name, email) VALUES (?, ?);
SELECT * FROM users WHERE id = LAST_INSERT_ID();

-- PUT with separate select
UPDATE users SET name = ?, email = ? WHERE id = ?;
SELECT * FROM users WHERE id = ?;
```

### SQLite

```sql
-- GET with relations
SELECT name, email FROM users WHERE id = ?;

-- POST with last_insert_rowid()
INSERT INTO users (name, email) VALUES (?, ?);
SELECT * FROM users WHERE id = last_insert_rowid();

-- PUT with separate select
UPDATE users SET name = ?, email = ? WHERE id = ?;
SELECT * FROM users WHERE id = ?;
```

### SurrealDB

```sql
-- GET with relations
SELECT name, email FROM users:123;

-- POST with RETURN
CREATE users SET name = $p1, email = $p2 RETURN *;

-- PUT with RETURN
UPDATE users:123 SET name = $p1, email = $p2 RETURN *;

-- PATCH with MERGE
UPDATE users:123 MERGE email = $p1 RETURN *;
```

## Security Considerations

All adapters implement:

- Parameterized queries to prevent SQL injection
- Input validation for all parameters
- Proper error handling and reporting
- Safe handling of database credentials

## Testing

Each adapter includes comprehensive tests covering:

- Query building for all HTTP methods
- Parameter binding and escaping
- Relations and filters handling
- Error validation
- Edge cases and special characters
- JSON handling and type conversions

## Error Handling

All adapters provide consistent error handling for:

- Missing required parameters (ID for PUT/PATCH/DELETE)
- Invalid JSON in request body
- Unsupported HTTP methods
- Invalid parameter types
- Database-specific syntax errors
