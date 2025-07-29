package responses

// GasFeeResult contains the result of gas fee calculation
type GasFeeResult struct {
	EstimatedGasUnits    uint64  `json:"estimated_gas_units"`
	GasPriceWei          string  `json:"gas_price_wei"`
	TotalGasCostWei      string  `json:"total_gas_cost_wei"`
	TotalGasCostEth      float64 `json:"total_gas_cost_eth"`
	TotalGasCostUSD      float64 `json:"total_gas_cost_usd"`
	TotalGasCostUSDCents int64   `json:"total_gas_cost_usd_cents"`
	Confidence           float64 `json:"confidence"` // 0.0 to 1.0
}

// EstimateGasFeeResult contains the result of gas fee estimation
type EstimateGasFeeResult struct {
	NetworkName           string  `json:"network_name"`
	TransactionType       string  `json:"transaction_type"`
	EstimatedGasUnits     uint64  `json:"estimated_gas_units"`
	CurrentGasPriceWei    string  `json:"current_gas_price_wei"`
	EstimatedCostWei      string  `json:"estimated_cost_wei"`
	EstimatedCostEth      float64 `json:"estimated_cost_eth"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
	EstimatedCostUSDCents int64   `json:"estimated_cost_usd_cents"`
	Confidence            float64 `json:"confidence"`
}
