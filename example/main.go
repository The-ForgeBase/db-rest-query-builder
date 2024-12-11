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
	db, err := sql.Open("sqlite://test.db")
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
	restQl := restql.NewRestQl("sqlite")
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {

		tb := getTable(r.URL.Path[5:])
		if tb == nil {
			http.Error(w, fmt.Sprintf("table %s not found", r.URL.Path[5:]), http.StatusNotFound)
			return
		}

		query, err := restQl.GetQL(r.URL.Path[5:], r, tb.PrimaryKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		rows, err := db.FetchData(r.Context(), query.Query, query.Args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// start server
	http.ListenAndServe(":8080", nil)
}
