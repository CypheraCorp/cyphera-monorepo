# Dashboard TODO - Cyphera API

This document tracks known issues, hidden features, and improvements needed for the Cyphera dashboard and subscription management system.

## 🚫 Currently Hidden/Disabled Features

These features exist in the codebase but are hidden from users because they don't work properly or return inaccurate data.

### Dashboard Metrics Issues

#### 1. **Churn Rate Calculations**
- **Status**: Hidden from UI
- **Issue**: Churn rate calculations are inaccurate or return 0%
- **Location**: `dashboard_metrics_service.go` churn calculation logic
- **Impact**: Cannot show customer retention metrics
- **Priority**: High

#### 2. **Cancelled Subscriptions Tracking**
- **Status**: Hidden from UI  
- **Issue**: Cancelled subscription counts are not properly tracked
- **Location**: Subscription status transitions and metrics calculation
- **Impact**: No visibility into subscription cancellations
- **Priority**: High

#### 3. **Annual Recurring Revenue (ARR)**
- **Status**: Hidden from UI
- **Issue**: ARR calculations may be inaccurate or inconsistent with MRR
- **Location**: `dashboard_metrics_service.go` ARR calculation
- **Impact**: Cannot show annual revenue projections
- **Priority**: Medium

#### 4. **Customer Analytics Graph**
- **Status**: Placeholder shown ("Customer analytics data being refined - coming soon")
- **Issue**: Customer growth, churn, and trend data is unreliable
- **Location**: `CustomerChart` component and related analytics endpoints
- **Impact**: No customer growth visibility
- **Priority**: Medium

#### 5. **Payment Status Distribution**
- **Status**: Hidden from UI
- **Issue**: Payment success/failure breakdown charts don't work properly
- **Location**: Payment analytics components and services
- **Impact**: Cannot monitor payment health
- **Priority**: Medium

#### 6. **Payment Volume Analytics**
- **Status**: Hidden from UI
- **Issue**: Payment volume over time charts are inaccurate
- **Location**: Payment volume calculation and charting components
- **Impact**: No payment trend visibility
- **Priority**: Low

## 📋 Subscription Management Issues

### 1. **Missing Term/Interval Information** ✅ FIXED
- **Status**: ~~Active bug~~ **RESOLVED**
- **Issue**: ~~Subscription list doesn't display billing intervals (monthly, yearly, etc.)~~ **FIXED**
- **Location**: ~~Subscription list components and API responses~~ **IMPLEMENTED**
- **Impact**: ~~Users can't see subscription frequency~~ **Now displays billing intervals**
- **Priority**: ~~High~~ **COMPLETED**
- **Details**: ~~Need to show "Monthly", "Yearly", "Weekly" etc. in subscription cards/tables~~ **Added billing interval column with formatted display**
- **Resolution**: 
  - Created `formatBillingInterval()` helper function in `/apps/web-app/src/lib/utils/format/billing.ts`
  - Added "Billing" column to merchants subscription list table
  - Displays intervals like "Monthly", "Every 3 months", "Yearly", etc.
  - Handles edge cases and various interval types from backend

## 🔧 Technical Debt & Improvements Needed

### Backend Metrics Calculation
- [ ] Fix churn rate calculation algorithm
- [ ] Implement proper subscription cancellation tracking
- [ ] Validate ARR vs MRR calculation consistency
- [ ] Add customer lifecycle tracking
- [ ] Improve payment status transition handling
- [ ] Add metrics calculation retry logic for timing issues

### Frontend Dashboard
- [ ] Re-enable churn rate display once backend is fixed
- [ ] Add cancelled subscriptions metrics
- [ ] Implement customer analytics graphs
- [ ] Create payment status distribution charts
- [ ] Add payment volume trend analysis
- [ ] Improve real-time metrics updates

### Subscription Management
- [ ] Add billing interval display to subscription lists
- [ ] Show next billing date information
- [ ] Add subscription frequency filtering
- [ ] Implement subscription term modification

## 🎯 Prioritized Action Items

### Immediate (Week 1)
1. ~~**Fix subscription interval display**~~ ✅ **COMPLETED** - Added billing interval column to subscription lists
2. ~~**Validate total revenue calculation**~~ ✅ **COMPLETED** - Fixed $0.00 total revenue timing issue
3. **Test MRR accuracy** - Verify Monthly Recurring Revenue calculations

### Short Term (2-4 weeks)
1. **Fix churn rate calculations** - Implement proper customer churn tracking
2. **Enable cancelled subscription metrics** - Track and display cancellation data
3. **Customer analytics restoration** - Fix customer growth/trend graphs

### Medium Term (1-2 months)
1. **Payment analytics suite** - Restore payment status and volume charts
2. **ARR calculation verification** - Ensure ARR accuracy and consistency
3. **Real-time metrics** - Improve dashboard refresh and calculation triggers

### Long Term (3+ months)
1. **Advanced analytics** - Customer lifetime value, cohort analysis
2. **Predictive metrics** - Churn prediction, revenue forecasting
3. **Comparative analytics** - Period-over-period comparisons

## 🧪 Testing Requirements

### Before Re-enabling Features
- [ ] Unit tests for all metrics calculations
- [ ] Integration tests with real subscription data
- [ ] End-to-end dashboard functionality tests
- [ ] Performance testing for large datasets
- [ ] Timezone handling verification

### Quality Gates
- Metrics must be accurate within 1% margin of error
- Dashboard must load within 2 seconds
- Real-time updates must work consistently
- No data inconsistencies between different views

## 📚 Documentation Needed

- [ ] Metrics calculation methodology documentation
- [ ] Dashboard feature user guide
- [ ] API documentation for analytics endpoints
- [ ] Troubleshooting guide for metrics issues

## 🔍 Investigation Notes

### Recent Findings
- **Total Revenue Issue**: Fixed timing issue where metrics calculated before payments committed
- **Data Consistency**: MRR/ARR show real data while total revenue was $0 - suggests different calculation paths
- **Database State**: Payments exist with correct 'completed' status, metrics calculation logic works in isolation

### Root Cause Patterns
- **Timing Issues**: Metrics calculations running before data fully processed
- **Transaction Isolation**: Calculations may not see uncommitted data
- **Status Transitions**: Payment/subscription status changes not properly tracked
- **Date Range Filtering**: Strict date filters may miss edge cases

---

**Last Updated**: July 31, 2025  
**Next Review**: August 7, 2025

## 📈 Recent Progress (July 31, 2025)
- ✅ **Fixed total revenue $0.00 issue**: Resolved timing problem in metrics calculation 
- ✅ **Added subscription billing intervals**: Merchants can now see subscription frequency (Monthly, Yearly, etc.)
- ✅ **Created billing format utilities**: Reusable functions for displaying billing periods

*This document should be updated as issues are resolved and new problems are discovered.*