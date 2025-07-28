package helpers

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds the parsed pagination parameters
type PaginationParams struct {
	Limit  int32
	Offset int32
	Page   int32
}

// ParsePaginationParams parses and validates pagination parameters from gin context
// Supports both page-based (?page=1&limit=10) and offset-based (?offset=0&limit=10) pagination
// Returns limit, offset, and page with safe defaults and validation
func ParsePaginationParams(c *gin.Context) (PaginationParams, error) {
	const maxLimit int32 = 100
	const defaultLimit int32 = 10
	const defaultOffset int32 = 0
	const defaultPage int32 = 1

	params := PaginationParams{
		Limit:  defaultLimit,
		Offset: defaultOffset,
		Page:   defaultPage,
	}

	// Parse limit parameter
	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := SafeParseInt32(limitStr)
		if err != nil {
			return params, fmt.Errorf("invalid limit parameter: %w", err)
		}
		if parsedLimit > 0 {
			if parsedLimit > maxLimit {
				params.Limit = maxLimit
			} else {
				params.Limit = parsedLimit
			}
		}
	}

	// Parse page parameter (for page-based pagination)
	if pageStr := c.Query("page"); pageStr != "" {
		parsedPage, err := SafeParseInt32(pageStr)
		if err != nil {
			return params, fmt.Errorf("invalid page parameter: %w", err)
		}
		if parsedPage > 0 {
			params.Page = parsedPage
			// Convert page to offset: offset = (page - 1) * limit
			params.Offset = (parsedPage - 1) * params.Limit
		}
	} else if offsetStr := c.Query("offset"); offsetStr != "" {
		// Parse offset parameter (for offset-based pagination)
		parsedOffset, err := SafeParseInt32(offsetStr)
		if err != nil {
			return params, fmt.Errorf("invalid offset parameter: %w", err)
		}
		if parsedOffset >= 0 {
			params.Offset = parsedOffset
			// Convert offset to page: page = (offset / limit) + 1
			params.Page = (parsedOffset / params.Limit) + 1
		}
	}

	return params, nil
}

// ParsePaginationParamsAsInt is a convenience function that returns regular int types
// for handlers that expect int instead of int32
func ParsePaginationParamsAsInt(c *gin.Context) (limit, offset int, err error) {
	params, err := ParsePaginationParams(c)
	if err != nil {
		return 0, 0, err
	}
	return int(params.Limit), int(params.Offset), nil
}

// SafeParseInt32 safely parses a string to int32, checking for overflow
func SafeParseInt32(s string) (int32, error) {
	// Parse as int64 first to check for overflow
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	// Check if value fits in int32
	if val > math.MaxInt32 || val < math.MinInt32 {
		return 0, fmt.Errorf("value %d overflows int32", val)
	}

	return int32(val), nil
}
