package business

// ChartDataPoint represents a single data point for charts
type ChartDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
	Label string  `json:"label,omitempty"`
}

// ChartData represents data for various chart types
type ChartData struct {
	ChartType string           `json:"chart_type"`
	Title     string           `json:"title"`
	Data      []ChartDataPoint `json:"data"`
	Period    string           `json:"period"`
}

// PieChartData represents data for pie charts
type PieChartData struct {
	ChartType string              `json:"chart_type"`
	Title     string              `json:"title"`
	Data      []PieChartDataPoint `json:"data"`
	Total     MoneyAmount         `json:"total"`
}

// PieChartDataPoint represents a single pie slice
type PieChartDataPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color,omitempty"`
}
