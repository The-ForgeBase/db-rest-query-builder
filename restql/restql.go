package restql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/The-ForgeBase/restql/log"
	"github.com/The-ForgeBase/restql/sql"
)

type RestQl struct {
	DriverName string `json:"driver_name"`
}

func NewRestQl(driverName string) *RestQl {
	return &RestQl{
		DriverName: driverName,
	}
}

func (s *RestQl) GetQL(tableName string, r *http.Request, primaryKey string) (*RestQlQuery, error) {

	if s.DriverName == "" {
		return nil, fmt.Errorf("driver name is empty")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name is empty")
	}

	// check table name
	pk := ""
	parts := strings.Split(tableName, "/")
	if len(parts) == 2 {
		tableName, pk = parts[0], parts[1]
	}

	urlQuery := sql.NewURLQuery(r.URL.Query(), s.DriverName)

	// check primary key
	if pk != "" {
		urlQuery.Set(primaryKey, fmt.Sprintf("eq.%s", pk))
		urlQuery.Set("singular", "")
	}

	var data *RestQlQuery
	switch r.Method {
	case "POST":
		d, err := s.create(r, tableName, urlQuery)
		if err != nil {
			return nil, err
		}
		data = d
	case "DELETE":
		d, err := s.delete(r, tableName, urlQuery)
		if err != nil {
			return nil, err
		}
		data = d
	case "PUT", "PATCH":
		d, err := s.update(r, tableName, urlQuery)
		if err != nil {
			return nil, err
		}
		data = d
	case "GET":
		d, err := s.get(r, tableName, urlQuery)
		if err != nil {
			return nil, err
		}
		data = d
	default:
		return nil, fmt.Errorf("method %s is not supported", r.Method)
	}

	return data, nil

}

func (s *RestQl) create(r *http.Request, tableName string, urlQuery *sql.URLQuery) (*RestQlQuery, error) {
	var data sql.PostData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Warnf("failed to parse post json data: %v", err)
		return nil, fmt.Errorf("failed to parse post json data, %v", err)
	}

	valuesQuery, err := data.ValuesQuery()
	if err != nil {
		log.Warnf("failed to generate values query %v", err)
		return nil, fmt.Errorf("failed to prepare values query, %v", err)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(valuesQuery.Columns, ","),
		strings.Join(valuesQuery.Placeholders, ","))
	args := valuesQuery.Args

	return s.returnQuery(query, args...), nil

}

func (s *RestQl) update(r *http.Request, tableName string, urlQuery *sql.URLQuery) (*RestQlQuery, error) {

	var data sql.PostData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Warnf("failed to parse update json data: %v", err)
		return nil, fmt.Errorf("failed to parse update json data, %v", err)
	}
	setQuery, err := data.SetQuery(1)
	if err != nil {
		log.Warnf("failed to generate set query: %v", err)
		return nil, fmt.Errorf("failed to prepare set query, %v", err)
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf("UPDATE %s SET %s", tableName, setQuery.Query))

	args := setQuery.Args
	_, whereQuery, args2 := urlQuery.WhereQuery(setQuery.Index)
	if whereQuery != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereQuery)
		args = append(args, args2...)
	} else {
		return nil, fmt.Errorf(`update without any condition is not allowed, please check the url query
If you really want to do it, uses 1=eq.1 to bypass it`)
	}

	query := queryBuilder.String()

	return s.returnQuery(query, args...), nil
}

func (s *RestQl) delete(r *http.Request, tableName string, urlQuery *sql.URLQuery) (*RestQlQuery, error) {

	var queryBuilder strings.Builder
	queryBuilder.WriteString("DELETE FROM ")
	queryBuilder.WriteString(tableName)
	_, whereQuery, args := urlQuery.WhereQuery(1)
	if whereQuery != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereQuery)
	} else {
		return nil, fmt.Errorf(`delete without any condition is not allowed, please check the url query
If you really want to do it, uses 1=eq.1 to bypass it`)
	}

	query := queryBuilder.String()

	return s.returnQuery(query, args...), nil
}

func (s *RestQl) get(r *http.Request, tableName string, urlQuery *sql.URLQuery) (*RestQlQuery, error) {

	if urlQuery.IsCount() {
		return s.count(r, tableName, urlQuery)
	}

	var queryBuilder strings.Builder
	selects, err := urlQuery.SelectQuery()
	if err != nil {
		log.Errorf("invalid select query %v", urlQuery)
		return nil, fmt.Errorf("invalid select query %v", urlQuery)
	}
	queryBuilder.WriteString(fmt.Sprintf("SELECT %s FROM %s", selects, tableName))
	_, whereQuery, args := urlQuery.WhereQuery(1)
	if whereQuery != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereQuery)
	}

	// order
	order := urlQuery.OrderQuery()
	if len(order) > 0 {
		queryBuilder.WriteString(" ORDER BY ")
		queryBuilder.WriteString(order)
	}

	// page operation
	page, pageSize := urlQuery.Page()
	queryBuilder.WriteString(" LIMIT ")
	queryBuilder.WriteString(fmt.Sprintf("%d", pageSize))
	if page != 1 {
		queryBuilder.WriteString(" OFFSET ")
		queryBuilder.WriteString(fmt.Sprintf("%d", (page-1)*pageSize))
	}

	query := queryBuilder.String()

	return s.returnQuery(query, args...), nil
}

func (s *RestQl) count(r *http.Request, tableName string, urlQuery *sql.URLQuery) (*RestQlQuery, error) {
	query := fmt.Sprintf("SELECT COUNT(1) AS count FROM %s", tableName)
	_, whereQuery, args := urlQuery.WhereQuery(1)
	if whereQuery != "" {
		query += fmt.Sprintf(" WHERE %s", whereQuery)
	}

	return s.returnQuery(query, args...), nil
}

func (s *RestQl) returnQuery(query string, args ...any) *RestQlQuery {
	return &RestQlQuery{
		Query: query,
		Args:  args,
	}
}

type RestQlQuery struct {
	Query string `json:"query"`
	Args  []any  `json:"args"`
}
