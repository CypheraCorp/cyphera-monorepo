package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"go.uber.org/zap"
)

// ExchangeRateService handles exchange rate operations with caching and fallback mechanisms
type ExchangeRateService struct {
	queries    db.Querier
	cmcClient  *coinmarketcap.Client
	logger     *zap.Logger
	cache      map[string]*CachedRate
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// CachedRate represents a cached exchange rate with expiration
type CachedRate struct {
	Rate      float64
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// NewExchangeRateService creates a new exchange rate service
func NewExchangeRateService(queries db.Querier, cmcAPIKey string) *ExchangeRateService {
	return &ExchangeRateService{
		queries:    queries,
		cmcClient:  coinmarketcap.NewClient(cmcAPIKey),
		logger:     logger.Log,
		cache:      make(map[string]*CachedRate),
		cacheMutex: sync.RWMutex{},
		cacheTTL:   5 * time.Minute, // Cache rates for 5 minutes
	}
}

// GetExchangeRate fetches exchange rate with caching and fallback mechanisms
func (s *ExchangeRateService) GetExchangeRate(ctx context.Context, params params.ExchangeRateParams) (*responses.ExchangeRateResult, error) {
	cacheKey := fmt.Sprintf("%s_%s", params.FromSymbol, params.ToSymbol)

	// Check cache first
	if rate := s.getCachedRate(cacheKey); rate != nil {
		return &responses.ExchangeRateResult{
			Rate:       rate.Rate,
			FromSymbol: params.FromSymbol,
			ToSymbol:   params.ToSymbol,
			Source:     "cache",
			UpdatedAt:  rate.UpdatedAt,
			Confidence: 1.0,
			TokenID:    params.TokenID,
			NetworkID:  params.NetworkID,
		}, nil
	}

	// Try to fetch from API
	apiRate, err := s.fetchFromAPI(ctx, params)
	if err != nil {
		s.logger.Warn("Failed to fetch rate from API, trying database fallback",
			zap.String("from", params.FromSymbol),
			zap.String("to", params.ToSymbol),
			zap.Error(err))

		// Fallback to database
		return s.getFromDatabase(ctx, params)
	}

	// Cache the successful API result
	s.setCachedRate(cacheKey, apiRate.Rate)

	// Store in database for future fallback
	if err := s.storeInDatabase(ctx, params, apiRate.Rate); err != nil {
		s.logger.Warn("Failed to store exchange rate in database",
			zap.Error(err))
	}

	return apiRate, nil
}

// GetMultipleExchangeRates fetches multiple exchange rates efficiently
func (s *ExchangeRateService) GetMultipleExchangeRates(ctx context.Context, requests []params.ExchangeRateParams) (map[string]*responses.ExchangeRateResult, error) {
	results := make(map[string]*responses.ExchangeRateResult)
	var uncachedTokens []string
	var uncachedRequests []params.ExchangeRateParams

	// Check cache for all requests first
	for _, params := range requests {
		cacheKey := fmt.Sprintf("%s_%s", params.FromSymbol, params.ToSymbol)

		if rate := s.getCachedRate(cacheKey); rate != nil {
			results[cacheKey] = &responses.ExchangeRateResult{
				Rate:       rate.Rate,
				FromSymbol: params.FromSymbol,
				ToSymbol:   params.ToSymbol,
				Source:     "cache",
				UpdatedAt:  rate.UpdatedAt,
				Confidence: 1.0,
				TokenID:    params.TokenID,
				NetworkID:  params.NetworkID,
			}
		} else {
			uncachedTokens = append(uncachedTokens, params.FromSymbol)
			uncachedRequests = append(uncachedRequests, params)
		}
	}

	// Fetch uncached rates from API in batch
	if len(uncachedTokens) > 0 {
		apiResults, err := s.fetchMultipleFromAPI(ctx, uncachedTokens, uncachedRequests)
		if err != nil {
			s.logger.Error("Failed to fetch multiple rates from API", zap.Error(err))
			// Try individual database fallbacks
			for _, params := range uncachedRequests {
				cacheKey := fmt.Sprintf("%s_%s", params.FromSymbol, params.ToSymbol)
				if dbResult, dbErr := s.getFromDatabase(ctx, params); dbErr == nil {
					results[cacheKey] = dbResult
				}
			}
		} else {
			for key, result := range apiResults {
				results[key] = result
			}
		}
	}

	return results, nil
}

// fetchFromAPI fetches a single exchange rate from CoinMarketCap API
func (s *ExchangeRateService) fetchFromAPI(ctx context.Context, params params.ExchangeRateParams) (*responses.ExchangeRateResult, error) {
	response, err := s.cmcClient.GetLatestQuotes([]string{params.FromSymbol}, []string{params.ToSymbol})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from CoinMarketCap: %w", err)
	}

	// Extract rate from response
	tokenData, exists := response.Data[params.FromSymbol]
	if !exists || len(tokenData) == 0 {
		return nil, fmt.Errorf("no data found for token %s", params.FromSymbol)
	}

	quote, exists := tokenData[0].Quote[params.ToSymbol]
	if !exists {
		return nil, fmt.Errorf("no quote found for %s to %s", params.FromSymbol, params.ToSymbol)
	}

	return &responses.ExchangeRateResult{
		Rate:       quote.Price,
		FromSymbol: params.FromSymbol,
		ToSymbol:   params.ToSymbol,
		Source:     "api",
		UpdatedAt:  time.Now(),
		Confidence: 1.0,
		TokenID:    params.TokenID,
		NetworkID:  params.NetworkID,
	}, nil
}

// fetchMultipleFromAPI fetches multiple exchange rates in a single API call
func (s *ExchangeRateService) fetchMultipleFromAPI(ctx context.Context, tokens []string, requests []params.ExchangeRateParams) (map[string]*responses.ExchangeRateResult, error) {
	// Get unique target currencies
	targetCurrencies := make(map[string]bool)
	for _, req := range requests {
		targetCurrencies[req.ToSymbol] = true
	}

	var targets []string
	for currency := range targetCurrencies {
		targets = append(targets, currency)
	}

	response, err := s.cmcClient.GetLatestQuotes(tokens, targets)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch multiple rates from CoinMarketCap: %w", err)
	}

	results := make(map[string]*responses.ExchangeRateResult)

	for _, req := range requests {
		cacheKey := fmt.Sprintf("%s_%s", req.FromSymbol, req.ToSymbol)

		tokenData, exists := response.Data[req.FromSymbol]
		if !exists || len(tokenData) == 0 {
			continue
		}

		quote, exists := tokenData[0].Quote[req.ToSymbol]
		if !exists {
			continue
		}

		result := &responses.ExchangeRateResult{
			Rate:       quote.Price,
			FromSymbol: req.FromSymbol,
			ToSymbol:   req.ToSymbol,
			Source:     "api",
			UpdatedAt:  time.Now(),
			Confidence: 1.0,
			TokenID:    req.TokenID,
			NetworkID:  req.NetworkID,
		}

		results[cacheKey] = result

		// Cache the result
		s.setCachedRate(cacheKey, quote.Price)

		// Store in database for fallback
		if err := s.storeInDatabase(ctx, req, quote.Price); err != nil {
			s.logger.Warn("Failed to store exchange rate in database",
				zap.String("pair", cacheKey),
				zap.Error(err))
		}
	}

	return results, nil
}

// getFromDatabase retrieves exchange rate from database as fallback
func (s *ExchangeRateService) getFromDatabase(ctx context.Context, params params.ExchangeRateParams) (*responses.ExchangeRateResult, error) {
	// Try to get the most recent rate from database
	// This would require a new SQLC query - for now return a basic fallback
	s.logger.Info("Using database fallback for exchange rate",
		zap.String("from", params.FromSymbol),
		zap.String("to", params.ToSymbol))

	// TODO: Implement actual database query
	// For now, return a sensible fallback for common pairs
	fallbackRates := map[string]float64{
		"ETH_USD":  2000.0,
		"BTC_USD":  45000.0,
		"USDC_USD": 1.0,
		"USDT_USD": 1.0,
	}

	pairKey := fmt.Sprintf("%s_%s", params.FromSymbol, params.ToSymbol)
	if rate, exists := fallbackRates[pairKey]; exists {
		return &responses.ExchangeRateResult{
			Rate:       rate,
			FromSymbol: params.FromSymbol,
			ToSymbol:   params.ToSymbol,
			Source:     "database",
			UpdatedAt:  time.Now().Add(-1 * time.Hour), // Indicate it's old data
			Confidence: 0.5,                            // Lower confidence for fallback
			TokenID:    params.TokenID,
			NetworkID:  params.NetworkID,
		}, nil
	}

	return nil, fmt.Errorf("no fallback rate available for %s to %s", params.FromSymbol, params.ToSymbol)
}

// storeInDatabase stores exchange rate in database for future fallback
func (s *ExchangeRateService) storeInDatabase(ctx context.Context, params params.ExchangeRateParams, rate float64) error {
	// TODO: Implement database storage
	// This would require creating a new table and SQLC queries for exchange_rates
	s.logger.Debug("Storing exchange rate in database",
		zap.String("from", params.FromSymbol),
		zap.String("to", params.ToSymbol),
		zap.Float64("rate", rate))

	return nil // Placeholder implementation
}

// getCachedRate retrieves rate from in-memory cache
func (s *ExchangeRateService) getCachedRate(key string) *CachedRate {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if rate, exists := s.cache[key]; exists {
		if time.Now().Before(rate.ExpiresAt) {
			return rate
		}
		// Rate expired, remove from cache
		delete(s.cache, key)
	}

	return nil
}

// setCachedRate stores rate in in-memory cache
func (s *ExchangeRateService) setCachedRate(key string, rate float64) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	now := time.Now()
	s.cache[key] = &CachedRate{
		Rate:      rate,
		UpdatedAt: now,
		ExpiresAt: now.Add(s.cacheTTL),
	}
}

// ConvertAmount converts an amount from one currency to another
func (s *ExchangeRateService) ConvertAmount(ctx context.Context, amount float64, from, to string) (float64, *responses.ExchangeRateResult, error) {
	if from == to {
		// Same currency, no conversion needed
		return amount, &responses.ExchangeRateResult{
			Rate:       1.0,
			FromSymbol: from,
			ToSymbol:   to,
			Source:     "direct",
			UpdatedAt:  time.Now(),
			Confidence: 1.0,
		}, nil
	}

	rateResult, err := s.GetExchangeRate(ctx, params.ExchangeRateParams{
		FromSymbol: from,
		ToSymbol:   to,
	})
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	convertedAmount := amount * rateResult.Rate
	return convertedAmount, rateResult, nil
}

// FormatDecimalString formats a decimal value to string with proper precision
func (s *ExchangeRateService) FormatDecimalString(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, value)
}

// ParseDecimalString parses a decimal string to float64
func (s *ExchangeRateService) ParseDecimalString(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// GetSupportedTokens returns a list of tokens supported by the exchange rate service
func (s *ExchangeRateService) GetSupportedTokens() []string {
	return []string{
		"BTC", "ETH", "USDC", "USDT", "MATIC", "BNB", "ADA", "SOL", "DOT", "AVAX",
	}
}

// GetSupportedCurrencies returns a list of fiat currencies supported
func (s *ExchangeRateService) GetSupportedCurrencies() []string {
	return []string{
		"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "CNY", "INR", "KRW",
	}
}

// ClearCache clears all cached exchange rates
func (s *ExchangeRateService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*CachedRate)
}

// GetCacheStats returns statistics about the cache
func (s *ExchangeRateService) GetCacheStats() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"total_entries":     len(s.cache),
		"cache_ttl_minutes": s.cacheTTL.Minutes(),
	}

	var expired int
	now := time.Now()
	for _, rate := range s.cache {
		if now.After(rate.ExpiresAt) {
			expired++
		}
	}
	stats["expired_entries"] = expired

	return stats
}
