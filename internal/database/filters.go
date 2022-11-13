package database

import (
	"math"
	"strings"

	"github.com/mroobert/json-api/internal/validator"
)

type (
	// Filters holds general applied filters
	// on a dataset from the database.
	Filters struct {
		Page         int
		PageSize     int
		Sort         string
		SortSafelist []string
	}

	// Metadata holds pagination metadata.
	Metadata struct {
		CurrentPage  int `json:"current_page,omitempty"`
		PageSize     int `json:"page_size,omitempty"`
		FirstPage    int `json:"first_page,omitempty"`
		LastPage     int `json:"last_page,omitempty"`
		TotalRecords int `json:"total_records,omitempty"`
	}
)

func NewMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}

// ValidateFilters checks if the filters are valid.
func (f Filters) ValidateFilters(v *validator.Validator) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// SortColumn computes the sorting column.
func (f Filters) SortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}

	panic("unsafe sort parameter: " + f.Sort)
}

// SortDirection computes the sorting direction.
func (f Filters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

// Limit represents maximum number of records that a SQL query should return.
func (f Filters) Limit() int {
	return f.PageSize
}

// Offset allows you to ‘skip’ a specific number of rows before starting to return
// records from the query.
func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}
