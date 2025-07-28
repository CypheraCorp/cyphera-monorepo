package business

// GasFeeMetrics represents aggregated gas fee metrics
type GasFeeMetrics struct {
	TotalTransactions   int64                      `json:"total_transactions"`
	TotalGasCostCents   int64                      `json:"total_gas_cost_cents"`
	SponsoredCostCents  int64                      `json:"sponsored_cost_cents"`
	AverageGasCostCents int64                      `json:"average_gas_cost_cents"`
	NetworkBreakdown    map[string]NetworkGasStats `json:"network_breakdown"`
}

// NetworkGasStats represents gas statistics for a specific network
type NetworkGasStats struct {
	Transactions int64 `json:"transactions"`
	CostCents    int64 `json:"cost_cents"`
	AvgCostCents int64 `json:"avg_cost_cents"`
}
