package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// AnalyticsHandler manages analytics and dashboard endpoints
type AnalyticsHandler struct {
	common  *CommonServices
	service *services.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(common *CommonServices) *AnalyticsHandler {
	// Get the pool from common services
	pool, err := common.GetDBPool()
	if err != nil {
		logger.Log.Fatal("Failed to get database pool for analytics handler", zap.Error(err))
	}

	analyticsService := services.NewAnalyticsService(common.db, pool)

	return &AnalyticsHandler{
		common:  common,
		service: analyticsService,
	}
}

// GetDashboardSummary returns the main dashboard metrics
// @Summary Get dashboard summary
// @Description Get key metrics for the dashboard overview
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param currency query string false "Currency code (default: workspace default)"
// @Success 200 {object} services.DashboardSummary
// @Router /api/v1/workspaces/{workspace_id}/analytics/dashboard [get]
func (h *AnalyticsHandler) GetDashboardSummary(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	currency := c.Query("currency")

	summary, err := h.service.GetDashboardSummary(c.Request.Context(), workspaceID, currency)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No metrics yet, trigger calculation
			go func() {
				ctx := context.Background()
				if err := h.service.TriggerMetricsRefresh(ctx, workspaceID, time.Now()); err != nil {
					logger.Log.Error("Failed to calculate metrics", zap.Error(err))
				}
			}()
			sendError(c, http.StatusNotFound, "No metrics available yet. Calculation in progress.", nil)
			return
		}
		handleDBError(c, err, "Failed to get dashboard metrics")
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetRevenueChart returns revenue data for charting
// @Summary Get revenue chart data
// @Description Get revenue data over time for charting
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param period query string false "Period: daily, weekly, monthly (default: daily)"
// @Param days query int false "Number of days to include (default: 30)"
// @Param currency query string false "Currency code"
// @Success 200 {object} services.ChartData
// @Router /api/v1/workspaces/{workspace_id}/analytics/revenue-chart [get]
func (h *AnalyticsHandler) GetRevenueChart(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	period := c.DefaultQuery("period", "daily")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	currency := c.Query("currency")

	chartData, err := h.service.GetRevenueChart(c.Request.Context(), workspaceID, period, days, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get revenue data")
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// GetCustomerChart returns customer growth data for charting
// @Summary Get customer chart data
// @Description Get customer metrics over time for charting
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param metric query string false "Metric: total, new, churned, growth_rate (default: total)"
// @Param period query string false "Period: daily, weekly, monthly (default: daily)"
// @Param days query int false "Number of days to include (default: 30)"
// @Success 200 {object} services.ChartData
// @Router /api/v1/workspaces/{workspace_id}/analytics/customer-chart [get]
func (h *AnalyticsHandler) GetCustomerChart(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	metric := c.DefaultQuery("metric", "total")
	period := c.DefaultQuery("period", "daily")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	currency := c.Query("currency")

	chartData, err := h.service.GetCustomerChart(c.Request.Context(), workspaceID, metric, period, days, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get customer data")
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// GetPaymentMetrics returns payment success and failure metrics
// @Summary Get payment metrics
// @Description Get payment success rates and volume metrics
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param days query int false "Number of days to include (default: 30)"
// @Success 200 {object} services.PaymentMetrics
// @Router /api/v1/workspaces/{workspace_id}/analytics/payment-metrics [get]
func (h *AnalyticsHandler) GetPaymentMetrics(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	currency := c.Query("currency")

	metrics, err := h.service.GetPaymentMetrics(c.Request.Context(), workspaceID, days, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get payment metrics")
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetNetworkBreakdown returns payment volume breakdown by network
// @Summary Get network breakdown
// @Description Get payment volume and count breakdown by blockchain network
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param date query string false "Date (YYYY-MM-DD, default: today)"
// @Success 200 {object} services.NetworkBreakdown
// @Router /api/v1/workspaces/{workspace_id}/analytics/network-breakdown [get]
func (h *AnalyticsHandler) GetNetworkBreakdown(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD", nil)
			return
		}
	} else {
		date = time.Now()
	}

	currency := c.Query("currency")

	breakdown, err := h.service.GetNetworkBreakdown(c.Request.Context(), workspaceID, date, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get network metrics")
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// RefreshMetrics triggers a recalculation of metrics
// @Summary Refresh metrics
// @Description Trigger a recalculation of dashboard metrics
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param date query string false "Date to recalculate (YYYY-MM-DD, default: today)"
// @Success 200 {object} MessageResponse
// @Router /api/v1/workspaces/{workspace_id}/analytics/refresh [post]
func (h *AnalyticsHandler) RefreshMetrics(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD", nil)
			return
		}
	} else {
		date = time.Now()
	}

	// Trigger async recalculation
	go func() {
		ctx := context.Background()
		if err := h.service.TriggerMetricsRefresh(ctx, workspaceID, date); err != nil {
			logger.Log.Error("Failed to refresh metrics",
				zap.String("workspace_id", workspaceID.String()),
				zap.Time("date", date),
				zap.Error(err),
			)
		}
	}()

	sendSuccessMessage(c, http.StatusOK, "Metrics refresh triggered successfully")
}

// GetSubscriptionChart returns subscription metrics for charting
// @Summary Get subscription chart data
// @Description Get subscription metrics over time for charting
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param metric query string false "Metric: active, new, cancelled, churn_rate (default: active)"
// @Param period query string false "Period: daily, weekly, monthly (default: daily)"
// @Param days query int false "Number of days to include (default: 30)"
// @Success 200 {object} services.ChartData
// @Router /api/v1/workspaces/{workspace_id}/analytics/subscription-chart [get]
func (h *AnalyticsHandler) GetSubscriptionChart(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	metric := c.DefaultQuery("metric", "active")
	period := c.DefaultQuery("period", "daily")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	currency := c.Query("currency")

	chartData, err := h.service.GetSubscriptionChart(c.Request.Context(), workspaceID, metric, period, days, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get subscription data")
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// GetMRRChart returns MRR/ARR growth over time
// @Summary Get MRR/ARR chart data
// @Description Get Monthly/Annual Recurring Revenue over time
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param metric query string false "Metric: mrr, arr (default: mrr)"
// @Param period query string false "Period: daily, weekly, monthly (default: monthly)"
// @Param months query int false "Number of months to include (default: 12)"
// @Success 200 {object} services.ChartData
// @Router /api/v1/workspaces/{workspace_id}/analytics/mrr-chart [get]
func (h *AnalyticsHandler) GetMRRChart(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	metric := c.DefaultQuery("metric", "mrr")
	period := c.DefaultQuery("period", "monthly")
	monthsStr := c.DefaultQuery("months", "12")
	months, _ := strconv.Atoi(monthsStr)
	if months <= 0 {
		months = 12
	}

	currency := c.Query("currency")

	chartData, err := h.service.GetMRRChart(c.Request.Context(), workspaceID, metric, period, months, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get MRR data")
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// GetGasFeePieChart returns gas fee breakdown as a pie chart
// @Summary Get gas fee pie chart
// @Description Get breakdown of gas fees (sponsored vs customer-paid)
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param days query int false "Number of days to include (default: 30)"
// @Success 200 {object} services.PieChartData
// @Router /api/v1/workspaces/{workspace_id}/analytics/gas-fee-pie [get]
func (h *AnalyticsHandler) GetGasFeePieChart(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	currency := c.Query("currency")

	pieChart, err := h.service.GetGasFeePieChart(c.Request.Context(), workspaceID, days, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get gas fee data")
		return
	}

	c.JSON(http.StatusOK, pieChart)
}

// GetHourlyMetrics returns hourly metrics for today
// @Summary Get hourly metrics
// @Description Get metrics broken down by hour for today
// @Tags Analytics
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {object} services.HourlyMetrics
// @Router /api/v1/workspaces/{workspace_id}/analytics/hourly [get]
func (h *AnalyticsHandler) GetHourlyMetrics(c *gin.Context) {
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	currency := c.Query("currency")

	metrics, err := h.service.GetHourlyMetrics(c.Request.Context(), workspaceID, currency)
	if err != nil {
		handleDBError(c, err, "Failed to get hourly metrics")
		return
	}

	c.JSON(http.StatusOK, metrics)
}