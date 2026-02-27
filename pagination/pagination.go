// Package pagination provides generic utilities for building paginated API
// responses. It includes a generic Response struct that wraps any item type
// with pagination metadata (page index, page size, total items, total pages),
// along with helper functions for calculating SQL offsets and total page counts.
package pagination

// Response represents a paginated API response with generic item type T.
// It wraps the actual data items together with pagination metadata that
// clients need to render pagination controls (page numbers, next/previous
// buttons, etc.).
//
// Type parameter T can be any type, but is typically a slice (e.g., []User)
// containing the items for the current page.
//
// JSON fields:
//   - items:       The actual data for the current page.
//   - page_index:  The current page number (1-based).
//   - page_size:   The maximum number of items per page.
//   - total_items: The total count of all items across all pages.
//   - total_pages: The total number of pages (calculated from total_items / page_size).
type Response[T any] struct {
	Items      T     `json:"items"`
	PageIndex  int   `json:"page_index"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// CalculateOffset computes the SQL OFFSET value from a 1-based page number
// and a page size (limit). The formula is (page - 1) * limit.
//
// This is intended for use with SQL queries that support LIMIT/OFFSET
// pagination, for example:
//
//	SELECT * FROM users ORDER BY id LIMIT $1 OFFSET $2
//
// Example:
//
//	offset := pagination.CalculateOffset(1, 20)  // 0  (first page)
//	offset = pagination.CalculateOffset(3, 20)   // 40 (third page)
func CalculateOffset(page, limit int) int {
	return (page - 1) * limit
}

// CalculateTotalPages computes the total number of pages needed to display
// all items, given the total item count and items per page (limit).
// It performs ceiling division: if there are any remaining items that
// don't fill a complete page, an extra page is added.
//
// Returns 0 if limit is 0 (to avoid division by zero).
//
// Example:
//
//	pages := pagination.CalculateTotalPages(100, 20)  // 5
//	pages = pagination.CalculateTotalPages(101, 20)   // 6 (ceiling division)
//	pages = pagination.CalculateTotalPages(0, 20)     // 0
func CalculateTotalPages(totalItems int64, limit int) int {
	if limit == 0 {
		return 0
	}

	totalPages := int(totalItems) / limit
	if int(totalItems)%limit > 0 {
		totalPages++
	}

	return totalPages
}

// NewResponse creates a new paginated Response by combining the items for the
// current page with the pagination metadata. It automatically calculates
// TotalPages using CalculateTotalPages.
//
// Parameters:
//   - items:      The data items for the current page (typically a slice).
//   - page:       The current 1-based page number.
//   - limit:      The maximum number of items per page (page size).
//   - totalItems: The total count of all matching items across all pages
//     (usually obtained from a COUNT(*) query).
//
// Example:
//
//	users := []User{ ... }                  // fetched from DB with LIMIT/OFFSET
//	var totalCount int64 = 95               // from COUNT(*) query
//	resp := pagination.NewResponse(users, 1, 20, totalCount)
//	// resp.TotalPages == 5, resp.PageIndex == 1, resp.PageSize == 20
func NewResponse[T any](items T, page, limit int, totalItems int64) Response[T] {
	return Response[T]{
		Items:      items,
		PageIndex:  page,
		PageSize:   limit,
		TotalItems: totalItems,
		TotalPages: CalculateTotalPages(totalItems, limit),
	}
}
