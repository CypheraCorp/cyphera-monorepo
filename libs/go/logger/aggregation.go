package logger

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/constants"
)

// LogAggregator provides functionality to aggregate and analyze log data
type LogAggregator struct {
	entries []LogEntry
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator() *LogAggregator {
	return &LogAggregator{
		entries: make([]LogEntry, 0),
	}
}

// AddEntry adds a log entry to the aggregator
func (la *LogAggregator) AddEntry(entry LogEntry) {
	la.entries = append(la.entries, entry)
}

// Aggregation result structures

// ErrorSummary contains error aggregation results
type ErrorSummary struct {
	TotalErrors       int            `json:"total_errors"`
	ErrorsByType      map[string]int `json:"errors_by_type"`
	ErrorsByComponent map[string]int `json:"errors_by_component"`
	TopErrors         []ErrorDetail  `json:"top_errors"`
	TimeRange         TimeRange      `json:"time_range"`
}

// ErrorDetail contains details about a specific error
type ErrorDetail struct {
	Message   string     `json:"message"`
	Component string     `json:"component"`
	Count     int        `json:"count"`
	LastSeen  time.Time  `json:"last_seen"`
	FirstSeen time.Time  `json:"first_seen"`
	Examples  []LogEntry `json:"examples,omitempty"`
}

// PerformanceSummary contains performance metrics
type PerformanceSummary struct {
	TotalOperations    int                         `json:"total_operations"`
	AverageLatency     time.Duration               `json:"average_latency"`
	P95Latency         time.Duration               `json:"p95_latency"`
	P99Latency         time.Duration               `json:"p99_latency"`
	SlowestOperations  []OperationMetrics          `json:"slowest_operations"`
	OperationBreakdown map[string]OperationMetrics `json:"operation_breakdown"`
	TimeRange          TimeRange                   `json:"time_range"`
}

// OperationMetrics contains metrics for a specific operation
type OperationMetrics struct {
	Operation     string        `json:"operation"`
	Count         int           `json:"count"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	SuccessRate   float64       `json:"success_rate"`
}

// ActivitySummary contains activity metrics
type ActivitySummary struct {
	TotalRequests       int            `json:"total_requests"`
	RequestsByEndpoint  map[string]int `json:"requests_by_endpoint"`
	RequestsByUser      map[string]int `json:"requests_by_user"`
	RequestsByComponent map[string]int `json:"requests_by_component"`
	HourlyBreakdown     map[string]int `json:"hourly_breakdown"`
	TimeRange           TimeRange      `json:"time_range"`
}

// LogAggregationQueries provides pre-built queries for common log analysis

// GetErrorSummary analyzes error patterns in the logs
func (la *LogAggregator) GetErrorSummary(timeRange TimeRange) ErrorSummary {
	summary := ErrorSummary{
		ErrorsByType:      make(map[string]int),
		ErrorsByComponent: make(map[string]int),
		TopErrors:         make([]ErrorDetail, 0),
		TimeRange:         timeRange,
	}

	errorDetails := make(map[string]*ErrorDetail)

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		// Only process error-level logs
		if entry.Level != constants.ErrorLevel {
			continue
		}

		summary.TotalErrors++

		// Count by component
		if entry.Component != "" {
			summary.ErrorsByComponent[entry.Component]++
		}

		// Categorize error type
		errorType := categorizeError(entry.Error)
		summary.ErrorsByType[errorType]++

		// Track error details
		key := fmt.Sprintf("%s:%s", entry.Component, entry.Message)
		if detail, exists := errorDetails[key]; exists {
			detail.Count++
			if entry.Timestamp.After(detail.LastSeen) {
				detail.LastSeen = entry.Timestamp
			}
			if entry.Timestamp.Before(detail.FirstSeen) {
				detail.FirstSeen = entry.Timestamp
			}
			if len(detail.Examples) < 3 {
				detail.Examples = append(detail.Examples, entry)
			}
		} else {
			errorDetails[key] = &ErrorDetail{
				Message:   entry.Message,
				Component: entry.Component,
				Count:     1,
				LastSeen:  entry.Timestamp,
				FirstSeen: entry.Timestamp,
				Examples:  []LogEntry{entry},
			}
		}
	}

	// Convert to sorted slice
	for _, detail := range errorDetails {
		summary.TopErrors = append(summary.TopErrors, *detail)
	}

	// Sort by count (descending)
	sort.Slice(summary.TopErrors, func(i, j int) bool {
		return summary.TopErrors[i].Count > summary.TopErrors[j].Count
	})

	// Limit to top 20
	if len(summary.TopErrors) > 20 {
		summary.TopErrors = summary.TopErrors[:20]
	}

	return summary
}

// GetPerformanceSummary analyzes performance metrics from the logs
func (la *LogAggregator) GetPerformanceSummary(timeRange TimeRange) PerformanceSummary {
	summary := PerformanceSummary{
		SlowestOperations:  make([]OperationMetrics, 0),
		OperationBreakdown: make(map[string]OperationMetrics),
		TimeRange:          timeRange,
	}

	durations := make([]time.Duration, 0)
	operationMetrics := make(map[string]*OperationMetrics)

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		// Only process entries with duration
		if entry.Duration <= 0 {
			continue
		}

		summary.TotalOperations++
		durations = append(durations, entry.Duration)

		// Track by operation
		if entry.Operation != "" {
			if metrics, exists := operationMetrics[entry.Operation]; exists {
				metrics.Count++
				metrics.TotalDuration += entry.Duration
				if entry.Duration > metrics.MaxDuration {
					metrics.MaxDuration = entry.Duration
				}
				if entry.Duration < metrics.MinDuration {
					metrics.MinDuration = entry.Duration
				}

				// Track success rate (assuming error level indicates failure)
				if entry.Level != constants.ErrorLevel {
					metrics.SuccessRate = float64(metrics.Count-1) / float64(metrics.Count) * metrics.SuccessRate
					metrics.SuccessRate += 1.0 / float64(metrics.Count)
				}
			} else {
				successRate := 1.0
				if entry.Level == constants.ErrorLevel {
					successRate = 0.0
				}

				operationMetrics[entry.Operation] = &OperationMetrics{
					Operation:     entry.Operation,
					Count:         1,
					TotalDuration: entry.Duration,
					MinDuration:   entry.Duration,
					MaxDuration:   entry.Duration,
					SuccessRate:   successRate,
				}
			}
		}
	}

	// Calculate statistics
	if len(durations) > 0 {
		// Sort durations for percentile calculations
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		// Calculate average
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		summary.AverageLatency = total / time.Duration(len(durations))

		// Calculate percentiles
		p95Index := int(float64(len(durations)) * 0.95)
		p99Index := int(float64(len(durations)) * 0.99)
		if p95Index >= len(durations) {
			p95Index = len(durations) - 1
		}
		if p99Index >= len(durations) {
			p99Index = len(durations) - 1
		}

		summary.P95Latency = durations[p95Index]
		summary.P99Latency = durations[p99Index]
	}

	// Calculate average durations for operations
	for operation, metrics := range operationMetrics {
		metrics.AvgDuration = metrics.TotalDuration / time.Duration(metrics.Count)
		summary.OperationBreakdown[operation] = *metrics
		summary.SlowestOperations = append(summary.SlowestOperations, *metrics)
	}

	// Sort slowest operations
	sort.Slice(summary.SlowestOperations, func(i, j int) bool {
		return summary.SlowestOperations[i].AvgDuration > summary.SlowestOperations[j].AvgDuration
	})

	// Limit to top 10
	if len(summary.SlowestOperations) > 10 {
		summary.SlowestOperations = summary.SlowestOperations[:10]
	}

	return summary
}

// GetActivitySummary analyzes activity patterns in the logs
func (la *LogAggregator) GetActivitySummary(timeRange TimeRange) ActivitySummary {
	summary := ActivitySummary{
		RequestsByEndpoint:  make(map[string]int),
		RequestsByUser:      make(map[string]int),
		RequestsByComponent: make(map[string]int),
		HourlyBreakdown:     make(map[string]int),
		TimeRange:           timeRange,
	}

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		summary.TotalRequests++

		// Count by component
		if entry.Component != "" {
			summary.RequestsByComponent[entry.Component]++
		}

		// Count by user
		if entry.UserID != "" {
			summary.RequestsByUser[entry.UserID]++
		}

		// Count by endpoint (extract from fields)
		if httpPath, exists := entry.Fields["http_path"]; exists {
			if path, ok := httpPath.(string); ok {
				summary.RequestsByEndpoint[path]++
			}
		}

		// Hourly breakdown
		hour := entry.Timestamp.Format("2006-01-02 15:00")
		summary.HourlyBreakdown[hour]++
	}

	return summary
}

// Utility functions

// categorizeError categorizes errors into types based on error message patterns
func categorizeError(errorMsg string) string {
	if errorMsg == "" {
		return "unknown"
	}

	errorMsg = strings.ToLower(errorMsg)

	patterns := map[string][]string{
		"database":   {"sql", "database", "connection", "query", "transaction", "deadlock"},
		"network":    {"network", "connection refused", "timeout", "dns", "unreachable"},
		"auth":       {"authentication", "authorization", "token", "credential", "permission"},
		"validation": {"validation", "invalid", "required", "format", "constraint"},
		"rate_limit": {"rate limit", "too many requests", "throttle"},
		"circle_api": {"circle", "wallet", "blockchain", "delegation"},
		"payment":    {"payment", "subscription", "billing", "charge"},
		"webhook":    {"webhook", "callback", "delivery"},
		"internal":   {"internal", "panic", "runtime", "nil pointer"},
	}

	for category, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(errorMsg, keyword) {
				return category
			}
		}
	}

	return "other"
}

// Common Query Patterns

// GetSlowQueries returns queries/operations that took longer than the threshold
func (la *LogAggregator) GetSlowQueries(threshold time.Duration, timeRange TimeRange) []LogEntry {
	slowQueries := make([]LogEntry, 0)

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		if entry.Duration > threshold {
			slowQueries = append(slowQueries, entry)
		}
	}

	// Sort by duration (slowest first)
	sort.Slice(slowQueries, func(i, j int) bool {
		return slowQueries[i].Duration > slowQueries[j].Duration
	})

	return slowQueries
}

// GetErrorsForUser returns all errors for a specific user
func (la *LogAggregator) GetErrorsForUser(userID string, timeRange TimeRange) []LogEntry {
	errors := make([]LogEntry, 0)

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		if entry.UserID == userID && entry.Level == constants.ErrorLevel {
			errors = append(errors, entry)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Timestamp.After(errors[j].Timestamp)
	})

	return errors
}

// GetCorrelatedLogs returns all logs with the same correlation ID
func (la *LogAggregator) GetCorrelatedLogs(correlationID string) []LogEntry {
	correlatedLogs := make([]LogEntry, 0)

	for _, entry := range la.entries {
		if entry.CorrelationID == correlationID {
			correlatedLogs = append(correlatedLogs, entry)
		}
	}

	// Sort by timestamp
	sort.Slice(correlatedLogs, func(i, j int) bool {
		return correlatedLogs[i].Timestamp.Before(correlatedLogs[j].Timestamp)
	})

	return correlatedLogs
}

// SearchLogs searches logs using regex patterns
func (la *LogAggregator) SearchLogs(pattern string, timeRange TimeRange, maxResults int) ([]LogEntry, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	results := make([]LogEntry, 0)

	for _, entry := range la.entries {
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		// Search in message and error fields
		if regex.MatchString(entry.Message) || regex.MatchString(entry.Error) {
			results = append(results, entry)

			if maxResults > 0 && len(results) >= maxResults {
				break
			}
		}
	}

	return results, nil
}

// Export functions for different formats

// ToJSON exports aggregation results as JSON-compatible structures
func (es ErrorSummary) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"total_errors":        es.TotalErrors,
		"errors_by_type":      es.ErrorsByType,
		"errors_by_component": es.ErrorsByComponent,
		"top_errors":          es.TopErrors,
		"time_range": map[string]interface{}{
			"start": es.TimeRange.Start,
			"end":   es.TimeRange.End,
		},
	}
}

// ToJSON exports performance summary as JSON-compatible structure
func (ps PerformanceSummary) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"total_operations":    ps.TotalOperations,
		"average_latency":     ps.AverageLatency.String(),
		"p95_latency":         ps.P95Latency.String(),
		"p99_latency":         ps.P99Latency.String(),
		"slowest_operations":  ps.SlowestOperations,
		"operation_breakdown": ps.OperationBreakdown,
		"time_range": map[string]interface{}{
			"start": ps.TimeRange.Start,
			"end":   ps.TimeRange.End,
		},
	}
}
