# RESTQL

RESTQL is a flexible Go package designed to generate SQL queries for SurrealDB, PostgreSQL, MySQL, and SQLite databases. It allows you to convert RESTful HTTP requests into native database queries, supporting dynamic table handling, filtering, pagination, sorting, and bulk operations like insertions, updates, and deletions.

The package dynamically builds SQL queries based on HTTP request parameters, making it simple to handle complex database interactions through simple REST APIs.

## Documentation

[RESTQL](https://docs.routechnology.org/s/afa79f7b-7b7a-4593-81b7-b66cbb503260/doc/restql-JqDwtDYiVT)

## Features

- **Dynamic Query Generation**: Generate SQL queries dynamically based on table names, filters, sorting, pagination, and more.
- **Filter Support**: Filter data with various operators, including `eq`, `ne`, `gt`, `lt`, `gte`, `lte`, and support for logical conditions such as `and` and `or`.
- **Pagination & Sorting**: Handle pagination with `page` and `page_size` parameters and sorting using the `order` parameter.
- **Bulk Operations**: Efficiently handle bulk insertions, updates, and deletions.
- **SurrealDB Support**: Full support for SurrealDB in addition to other databases (PostgreSQL, MySQL, SQLite).

## Installation

To install the package, run:

```bash
go get github.com/The-ForgeBase/restql
```

## Usage

Here’s a simple example using RESTQL with SurrealDB. This example sets up a REST API server that generates SQL queries based on HTTP requests.

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/The-ForgeBase/restql/log"
	"github.com/The-ForgeBase/restql/restql"
	"github.com/The-ForgeBase/restql/sql"
)

var (
	tablesMu sync.RWMutex
	tables   map[string]*sql.Table
)

func getTable(tableName string) *sql.Table {
	tablesMu.RLock()
	defer tablesMu.RUnlock()
	return tables[tableName]
}

func main() {
	// open db
	db, err := sql.Open("surrealdb://localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// fetch all tables
	tb := db.FetchTables()

	tablesMu.Lock()
	tables = tb
	tablesMu.Unlock()

	// create restql handler
	restQl := restql.NewRestQl("surrealdb")
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {

		// Generate query using RESTQL
		query, err := restQl.GetQL(r, "surrealdb")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Fetch data using generated query for SurrealDB
		// just use the query as it is without using the args

		// Fetch data using generated query for SQL
		rows, err := db.FetchData(r.Context(), query.Query, query.Args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Encode rows as JSON response
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Start server
	http.ListenAndServe(":8080", nil)
}
```

## HTTP Query Parameters

### Filtering

Support for a variety of comparison operators to filter data:

- `eq` (equals), `ne` (not equals), `gt` (greater than), `gte` (greater than or equal), `lt` (less than), `lte` (less than or equal).
- Example: `/products?level=eq.2`

### Logical Operators

Combine multiple filters using `and` and `or`:

- Example: `/products?level=lt.2&hidden=is.false`
- Example: `/products?or=(level=lt.2,hidden=is.false)`

### Pagination & Sorting

Support for pagination and sorting:

- Use `page` and `page_size` for pagination.
- Use `order` for sorting (e.g., `order=level.asc`).
- Example: `/products?page=2&page_size=10&order=level.asc`

### Bulk Operations

Supports bulk insertions, updates, and deletions:

- **POST**: Insert one or more records into a table.
- **PUT**: Update records by primary key or filters.
- **DELETE**: Delete records by primary key or using filters.

## Example Queries

1. **GET Request with Filters and Pagination**:

   - URL: `/products?level=gte.10&order=price.desc&page=1&page_size=20`
   - SQL: `SELECT * FROM products WHERE level >= ? ORDER BY price DESC LIMIT 20 OFFSET 0`

2. **POST Request for Bulk Insert**:

   - URL: `/products`
   - Body:
     ```json
     [
       { "name": "Product A", "price": 100 },
       { "name": "Product B", "price": 150 }
     ]
     ```
   - SQL: `INSERT INTO products (name, price) VALUES (?, ?), (?, ?)`

3. **PUT Request for Bulk Update**:

   - URL: `/products/1`
   - Body:
     ```json
     { "name": "Updated Product", "price": 200 }
     ```
   - SQL: `UPDATE products SET name = ?, price = ? WHERE id = ?`

4. **DELETE Request with Filters**:
   - URL: `/products?level=lt.10`
   - SQL: `DELETE FROM products WHERE level < ?`

## Test Coverage

The package is fully tested for all methods and features. Below are some of the key scenarios tested:

- **GET Request Validation**: Ensures correct handling of table names, methods, and query generation.
- **Filtering and Pagination**: Tests various filters (`eq`, `lt`, `gt`, etc.), pagination (`page`, `page_size`), and sorting (`order`).
- **Bulk Insertions**: Verifies single and bulk insertions with correct SQL generation.
- **Updates**: Tests updating records by primary key and valid JSON input.
- **Deletes**: Verifies delete operations using both primary key and filters.

## SurrealDB Support

SurrealDB support has been fully implemented, and the package now works seamlessly with SurrealDB databases. Ensure you specify the correct database type (`surrealdb`) when initializing the `restql` handler.

## Contributions

Contributions to this project are welcome! Please feel free to fork the repository, create a pull request, and suggest improvements.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
