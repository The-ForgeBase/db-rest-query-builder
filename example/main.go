package main

import (
	"net/http"
)

// var (
// 	tablesMu sync.RWMutex
// 	tables   map[string]*sql.Table
// )

// func getTable(tableName string) *sql.Table {
// 	tablesMu.RLock()
// 	defer tablesMu.RUnlock()
// 	return tables[tableName]
// }

func main() {

	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {

	})

	// start server
	http.ListenAndServe(":8080", nil)
}
