# Frontend Implementation Plan - Part 2

## Overview

This document outlines the comprehensive frontend implementation plan for the Cyphera web application, building upon the backend enhancements described in the platform enhancement implementation plan. This plan focuses exclusively on the frontend changes needed to support the new backend capabilities.

## Current State Analysis

### Existing Strengths
- **Modern Tech Stack**: Next.js 15, TypeScript, React Query, Zustand
- **Component Architecture**: Well-organized with shadcn/ui components
- **State Management**: Clear separation between server state (React Query) and UI state (Zustand)
- **Type Safety**: Full TypeScript coverage with proper typing
- **API Integration**: Centralized API client with proper error handling

### Key Gaps to Address
1. **Static Dashboard**: Currently shows hardcoded $0.00 values
2. **No Invoice System**: Missing invoice creation, management, and display
3. **Limited Financial Views**: No revenue charts, analytics, or reports
4. **Basic Currency Support**: Hardcoded USD/EUR, no dynamic formatting
5. **No Tax Display**: No tax calculation or display in UI
6. **No Gas Fee Visibility**: Gas fees not shown to users
7. **Missing Payment Features**: No payment links, QR codes, or delegation UI

## Frontend Architecture Enhancements

### 1. Data Layer Architecture

#### Enhanced API Client
```typescript
// services/api/enhanced-client.ts
class EnhancedCypheraAPI extends CypheraAPI {
  // New endpoints for enhanced features
  dashboard = {
    getMetrics: (period: string) => this.get('/dashboard/metrics', { period }),
    getRevenueChart: (range: DateRange) => this.get('/dashboard/revenue', range),
    getGasAnalytics: () => this.get('/gas-sponsorship/analytics'),
  };

  invoices = {
    list: (params: ListParams) => this.get('/invoices', params),
    create: (data: CreateInvoiceDTO) => this.post('/invoices', data),
    get: (id: string) => this.get(`/invoices/${id}`),
    update: (id: string, data: UpdateInvoiceDTO) => this.put(`/invoices/${id}`, data),
    sendToCustomer: (id: string) => this.post(`/invoices/${id}/send`),
    getLineItems: (id: string) => this.get(`/invoices/${id}/line-items`),
  };

  payments = {
    list: (params: ListParams) => this.get('/payments', params),
    getByInvoice: (invoiceId: string) => this.get(`/payments/invoice/${invoiceId}`),
    getGasFees: (paymentId: string) => this.get(`/payments/${paymentId}/gas-fees`),
  };

  tax = {
    calculate: (items: LineItem[]) => this.post('/taxes/calculate', { items }),
    getJurisdictions: () => this.get('/taxes/jurisdictions'),
    validateTaxId: (taxId: string, country: string) => 
      this.post('/taxes/validate-tax-id', { taxId, country }),
  };

  currencies = {
    list: () => this.get('/currencies'),
    get: (code: string) => this.get(`/currencies/${code}`),
  };
}
```

#### Enhanced React Query Hooks
```typescript
// hooks/data/useDashboard.ts
export function useDashboardMetrics(period: 'day' | 'week' | 'month' | 'year') {
  const queryClient = useQueryClient();
  
  return useQuery({
    queryKey: ['dashboard', 'metrics', period],
    queryFn: () => api.dashboard.getMetrics(period),
    staleTime: 60 * 1000, // 1 minute
    refetchInterval: 5 * 60 * 1000, // 5 minutes
    
    // Optimistic update when payment completes
    onSuccess: (data) => {
      queryClient.setQueryData(['workspace', 'stats'], data.summary);
    },
  });
}

// Real-time updates via WebSocket
export function useRealtimeMetrics() {
  const queryClient = useQueryClient();
  
  useEffect(() => {
    const ws = new WebSocket(process.env.NEXT_PUBLIC_WS_URL);
    
    ws.on('payment.completed', (payment) => {
      // Invalidate affected queries
      queryClient.invalidateQueries(['dashboard']);
      queryClient.invalidateQueries(['payments']);
      queryClient.invalidateQueries(['invoices', payment.invoice_id]);
      
      // Show notification
      toast.success(`Payment received: ${formatCurrency(payment.amount)}`);
    });
    
    return () => ws.close();
  }, [queryClient]);
}

// hooks/finance/use-currency.ts
export function useCurrency() {
  const { data: currencies, isLoading } = useQuery({
    queryKey: ['fiat-currencies'],
    queryFn: async () => {
      const response = await fetch('/api/currencies');
      if (!response.ok) throw new Error('Failed to fetch currencies');
      return response.json();
    },
    staleTime: 1000 * 60 * 60, // Cache for 1 hour
  });
  
  const formatCurrency = useCallback((
    amountInCents: number,
    currencyCode: string = 'USD'
  ) => {
    const currency = currencies?.find(c => c.code === currencyCode);
    if (!currency) {
      // Fallback for unknown currencies
      return `${(amountInCents / 100).toFixed(2)} ${currencyCode}`;
    }
    
    const amount = amountInCents / Math.pow(10, currency.decimal_places);
    const formatted = new Intl.NumberFormat(currency.locale || 'en-US', {
      style: 'currency',
      currency: currency.code,
      minimumFractionDigits: currency.decimal_places,
      maximumFractionDigits: currency.decimal_places,
    }).format(amount);
    
    // Handle custom symbol positions if needed
    if (currency.symbol_position === 'after' && currency.symbol) {
      return formatted.replace(currency.code, '').trim() + currency.symbol;
    }
    
    return formatted;
  }, [currencies]);
  
  const getCurrencyData = useCallback((currencyCode: string) => {
    return currencies?.find(c => c.code === currencyCode) || null;
  }, [currencies]);
  
  const getSupportedCurrencies = useCallback(() => {
    return currencies?.filter(c => c.is_active) || [];
  }, [currencies]);
  
  return {
    currencies,
    isLoading,
    formatCurrency,
    getCurrencyData,
    getSupportedCurrencies,
  };
}
```

### 2. State Management Enhancements

#### New Zustand Stores
```typescript
// stores/financial.ts
interface FinancialUIStore {
  // Currency preferences
  displayCurrency: string;
  currencyData: FiatCurrency | null;
  setCurrency: (code: string) => void;
  
  // Dashboard preferences
  dashboardPeriod: 'day' | 'week' | 'month' | 'year';
  chartType: 'line' | 'bar' | 'area';
  
  // Invoice builder state
  invoiceBuilder: {
    lineItems: LineItem[];
    customer: Customer | null;
    taxCalculation: TaxCalculation | null;
    discounts: Discount[];
  };
  
  // Gas sponsorship preferences
  showGasSponsorship: boolean;
  gasDisplayMode: 'inline' | 'tooltip' | 'separate';
}

// stores/invoice.ts
interface InvoiceUIStore {
  // List view preferences
  viewMode: 'table' | 'cards';
  sortBy: string;
  filters: InvoiceFilters;
  
  // Builder state
  isBuilding: boolean;
  draftInvoice: Partial<Invoice>;
  validationErrors: Record<string, string>;
  
  // Actions
  startInvoiceBuilder: () => void;
  updateLineItem: (index: number, item: LineItem) => void;
  calculateTotals: () => InvoiceTotals;
}
```

### 3. Component Architecture

#### Dashboard Components

```typescript
// components/dashboard/MetricsDashboard.tsx
export function MetricsDashboard() {
  const { period } = useFinancialUI();
  const { data: metrics, isLoading } = useDashboardMetrics(period);
  const { formatCurrency } = useCurrency();
  
  if (isLoading) return <DashboardSkeleton />;
  
  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Monthly Recurring Revenue"
          value={formatCurrency(metrics.mrr_cents)}
          change={metrics.mrr_change}
          trend={metrics.mrr_trend}
          icon={TrendingUp}
        />
        <MetricCard
          title="Active Subscriptions"
          value={metrics.active_subscriptions}
          change={metrics.subscription_change}
          icon={Users}
        />
        <MetricCard
          title="Total Gas Sponsored"
          value={formatCurrency(metrics.sponsored_gas_fees_cents)}
          subtitle={`${metrics.gas_sponsorship_rate}% sponsorship rate`}
          icon={Fuel}
        />
        <MetricCard
          title="Tax Collected"
          value={formatCurrency(metrics.tax_collected_cents)}
          subtitle={`Across ${metrics.tax_jurisdictions} jurisdictions`}
          icon={Receipt}
        />
      </div>
      
      {/* Revenue Chart */}
      <RevenueChart data={metrics.chart_data} />
      
      {/* Recent Activity */}
      <div className="grid gap-6 lg:grid-cols-2">
        <RecentPayments />
        <PendingInvoices />
      </div>
    </div>
  );
}
```

#### Invoice System Components

```typescript
// components/invoices/InvoiceBuilder.tsx
export function InvoiceBuilder() {
  const { draftInvoice, updateLineItem } = useInvoiceUI();
  const { calculateTax } = useTaxCalculation();
  const { formatCurrency } = useCurrency();
  
  return (
    <div className="grid gap-6 lg:grid-cols-3">
      {/* Main Builder */}
      <div className="lg:col-span-2 space-y-6">
        <CustomerSelector />
        
        <Card>
          <CardHeader>
            <CardTitle>Line Items</CardTitle>
          </CardHeader>
          <CardContent>
            <LineItemsBuilder
              items={draftInvoice.lineItems}
              onUpdate={updateLineItem}
              onAdd={addLineItem}
              onRemove={removeLineItem}
            />
          </CardContent>
        </Card>
        
        <TaxConfiguration
          customer={draftInvoice.customer}
          onTaxCalculated={setTaxCalculation}
        />
        
        <GasFeeOptions
          estimatedGas={draftInvoice.estimatedGas}
          sponsorshipEnabled={workspace.gasSponsorshipEnabled}
        />
      </div>
      
      {/* Preview Sidebar */}
      <div className="space-y-6">
        <InvoicePreview
          invoice={draftInvoice}
          totals={calculateTotals()}
        />
        
        <Card>
          <CardContent className="space-y-4">
            <Button
              onClick={saveDraft}
              variant="outline"
              className="w-full"
            >
              Save Draft
            </Button>
            <Button
              onClick={finalizeAndSend}
              className="w-full"
              disabled={!isValid}
            >
              Finalize & Send
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// components/invoices/LineItemsBuilder.tsx
export function LineItemsBuilder({ items, onUpdate, onAdd, onRemove }) {
  return (
    <div className="space-y-4">
      {items.map((item, index) => (
        <LineItemRow
          key={item.id}
          item={item}
          index={index}
          onUpdate={(updates) => onUpdate(index, updates)}
          onRemove={() => onRemove(index)}
          showGasSeparately={item.type === 'product'}
        />
      ))}
      
      <div className="flex gap-2">
        <Button
          onClick={() => onAdd({ type: 'product' })}
          variant="outline"
          size="sm"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Product
        </Button>
        <Button
          onClick={() => onAdd({ type: 'custom' })}
          variant="outline"
          size="sm"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Custom Item
        </Button>
      </div>
    </div>
  );
}
```

#### Enhanced Currency Display

```typescript
// components/ui/currency-display.tsx
export function CurrencyDisplay({ 
  amountInCents, 
  currency,
  showCode = false,
  className 
}: CurrencyDisplayProps) {
  const { formatCurrency, getCurrencyData } = useCurrency();
  const currencyData = getCurrencyData(currency);
  
  if (!currencyData) {
    return <span className={className}>{amountInCents / 100}</span>;
  }
  
  const formatted = formatCurrency(amountInCents, currencyData);
  
  return (
    <span className={cn("font-mono", className)}>
      {formatted}
      {showCode && (
        <span className="text-muted-foreground ml-1">
          {currencyData.code}
        </span>
      )}
    </span>
  );
}

// components/ui/multi-currency-selector.tsx
export function MultiCurrencySelector({ 
  value,
  onChange,
  supportedCurrencies 
}: MultiCurrencySelectorProps) {
  const { data: currencies } = useCurrencies();
  
  const available = currencies?.filter(c => 
    supportedCurrencies.includes(c.code)
  );
  
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {available?.map(currency => (
          <SelectItem key={currency.code} value={currency.code}>
            <div className="flex items-center gap-2">
              <span className="font-mono">{currency.symbol}</span>
              <span>{currency.name}</span>
              <span className="text-muted-foreground">({currency.code})</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
```

#### Tax Display Components

```typescript
// components/tax/TaxBreakdown.tsx
export function TaxBreakdown({ 
  calculations,
  showJurisdictions = true 
}: TaxBreakdownProps) {
  if (!calculations || calculations.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">
        No tax applicable
      </div>
    );
  }
  
  return (
    <div className="space-y-2">
      {calculations.map(calc => (
        <div key={calc.id} className="flex justify-between text-sm">
          <span>
            {calc.tax_type}
            {showJurisdictions && (
              <span className="text-muted-foreground ml-1">
                ({calc.jurisdiction_name})
              </span>
            )}
          </span>
          <span>{formatCurrency(calc.amount_cents)}</span>
        </div>
      ))}
      
      {calculations.some(c => c.reverse_charge_applies) && (
        <Alert>
          <AlertDescription className="text-xs">
            Reverse charge applies - Customer is responsible for tax remittance
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

// components/tax/CustomerTaxInfo.tsx
export function CustomerTaxInfo({ customer }: { customer: Customer }) {
  const [isEditing, setIsEditing] = useState(false);
  const { validateTaxId } = useTaxValidation();
  
  if (!customer.is_business) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            Individual customer - Standard tax rates apply
          </p>
        </CardContent>
      </Card>
    );
  }
  
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Tax Information</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div>
            <Label className="text-xs">Business Name</Label>
            <p className="font-medium">{customer.business_name}</p>
          </div>
          
          <div>
            <Label className="text-xs">Tax ID</Label>
            <div className="flex items-center gap-2">
              <p className="font-mono">
                {customer.tax_id || 'Not provided'}
              </p>
              {customer.tax_id_verified && (
                <Badge variant="success" className="text-xs">
                  <CheckCircle className="h-3 w-3 mr-1" />
                  Verified
                </Badge>
              )}
            </div>
          </div>
          
          <div>
            <Label className="text-xs">Tax Jurisdiction</Label>
            <p>{customer.jurisdiction?.name || 'Not determined'}</p>
          </div>
          
          {customer.tax_exemptions?.length > 0 && (
            <div>
              <Label className="text-xs">Exemptions</Label>
              <div className="flex flex-wrap gap-1 mt-1">
                {customer.tax_exemptions.map(exemption => (
                  <Badge key={exemption.id} variant="secondary">
                    {exemption.type}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>
        
        <Button
          onClick={() => setIsEditing(true)}
          variant="outline"
          size="sm"
          className="mt-4 w-full"
        >
          Update Tax Information
        </Button>
      </CardContent>
    </Card>
  );
}
```

#### Gas Fee Components

```typescript
// components/gas/GasFeeDisplay.tsx
export function GasFeeDisplay({ 
  payment,
  displayMode = 'inline' 
}: GasFeeDisplayProps) {
  const { formatCurrency } = useCurrency();
  const gasData = payment.gas_fee_payment;
  
  if (!gasData) return null;
  
  const isSponsored = gasData.sponsor_type === 'merchant';
  
  if (displayMode === 'inline') {
    return (
      <span className="text-sm">
        {isSponsored ? (
          <span className="text-green-600">
            Gas sponsored
          </span>
        ) : (
          <span className="text-muted-foreground">
            + {formatCurrency(gasData.gas_fee_usd_cents)} gas
          </span>
        )}
      </span>
    );
  }
  
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="sm">
          <Fuel className="h-4 w-4" />
          {isSponsored && <Badge className="ml-1">Sponsored</Badge>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <div className="space-y-3">
          <h4 className="font-medium">Gas Fee Details</h4>
          
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span>Network</span>
              <span>{gasData.network_name}</span>
            </div>
            <div className="flex justify-between">
              <span>Gas Used</span>
              <span>{gasData.gas_units_used} units</span>
            </div>
            <div className="flex justify-between">
              <span>Gas Price</span>
              <span>{gasData.gas_price_gwei} gwei</span>
            </div>
            <Separator />
            <div className="flex justify-between font-medium">
              <span>Total Gas Fee</span>
              <span>{formatCurrency(gasData.gas_fee_usd_cents)}</span>
            </div>
            {isSponsored && (
              <div className="text-green-600 text-xs">
                This gas fee was paid by the merchant
              </div>
            )}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

// components/gas/GasSponsorshipSettings.tsx
export function GasSponsorshipSettings() {
  const { data: config, mutate: updateConfig } = useGasSponsorshipConfig();
  const { formatCurrency } = useCurrency();
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>Gas Fee Sponsorship</CardTitle>
        <CardDescription>
          Improve conversion by sponsoring customer gas fees
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <Label>Enable Sponsorship</Label>
            <p className="text-sm text-muted-foreground">
              Pay gas fees on behalf of your customers
            </p>
          </div>
          <Switch
            checked={config?.sponsorship_enabled}
            onCheckedChange={(enabled) => 
              updateConfig({ sponsorship_enabled: enabled })
            }
          />
        </div>
        
        {config?.sponsorship_enabled && (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <Label>Monthly Budget</Label>
                <Input
                  type="number"
                  value={config.monthly_budget_usd_cents / 100}
                  onChange={(e) => updateConfig({
                    monthly_budget_usd_cents: parseFloat(e.target.value) * 100
                  })}
                  placeholder="100.00"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Spent this month: {formatCurrency(config.current_month_spent_cents)}
                </p>
              </div>
              
              <div>
                <Label>Max Per Transaction</Label>
                <Input
                  type="number"
                  value={config.sponsor_threshold_usd_cents / 100}
                  onChange={(e) => updateConfig({
                    sponsor_threshold_usd_cents: parseFloat(e.target.value) * 100
                  })}
                  placeholder="5.00"
                />
              </div>
            </div>
            
            <GasSponsorshipRules config={config} onUpdate={updateConfig} />
            <GasSponsorshipAnalytics />
          </>
        )}
      </CardContent>
    </Card>
  );
}
```

#### Discount & Trial Components

```typescript
// components/discounts/DiscountCodeInput.tsx
export function DiscountCodeInput({ 
  onApply, 
  lineItems,
  disabled = false 
}: DiscountCodeInputProps) {
  const [code, setCode] = useState('');
  const [discount, setDiscount] = useState<Discount | null>(null);
  const [isValidating, setIsValidating] = useState(false);

  const handleApplyDiscount = async () => {
    setIsValidating(true);
    try {
      const response = await api.discounts.validate({
        code,
        line_items: lineItems,
        customer_id: getCurrentCustomerId(),
      });

      if (response.valid) {
        setDiscount(response.discount);
        onApply(response.discount);
        toast.success(`Discount applied: ${response.discount.description}`);
      } else {
        toast.error(response.error || 'Invalid discount code');
      }
    } catch (error) {
      toast.error('Failed to validate discount code');
    } finally {
      setIsValidating(false);
    }
  };

  const removeDiscount = () => {
    setDiscount(null);
    setCode('');
    onApply(null);
  };

  if (discount) {
    return (
      <div className="border rounded-lg p-4 bg-green-50 dark:bg-green-950">
        <div className="flex items-center justify-between">
          <div>
            <p className="font-medium text-green-800 dark:text-green-200">
              Discount Applied
            </p>
            <p className="text-sm text-green-600 dark:text-green-400">
              {discount.discount_type === 'percentage' && 
                `${discount.discount_value}% off`}
              {discount.discount_type === 'fixed_amount' && 
                `${formatCurrency(discount.discount_value)} off`}
              {discount.grants_trial_days > 0 && 
                ` + ${discount.grants_trial_days} day free trial`}
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={removeDiscount}
            disabled={disabled}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="border rounded-lg p-4">
      <Label htmlFor="discount-code" className="text-sm font-medium mb-2 block">
        Have a discount code?
      </Label>
      <div className="flex gap-2">
        <Input
          id="discount-code"
          placeholder="Enter code"
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          disabled={disabled || isValidating}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && code) {
              e.preventDefault();
              handleApplyDiscount();
            }
          }}
        />
        <Button
          onClick={handleApplyDiscount}
          disabled={!code || isValidating || disabled}
          variant="secondary"
        >
          {isValidating ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            'Apply'
          )}
        </Button>
      </div>
    </div>
  );
}

// components/subscriptions/SubscriptionPricingDisplay.tsx
export function SubscriptionPricingDisplay({ 
  subscription,
  lineItems,
  discount,
  trial,
  showBreakdown = true 
}: SubscriptionPricingDisplayProps) {
  const { formatCurrency } = useCurrency();
  
  const calculateDiscountedPrice = useCallback((item: LineItem) => {
    if (!discount) return item.amount_in_cents;
    
    if (discount.discount_type === 'percentage') {
      return Math.round(item.amount_in_cents * (1 - discount.discount_value / 100));
    } else if (discount.discount_type === 'fixed_amount') {
      // Distribute fixed discount proportionally across items
      const totalAmount = lineItems.reduce((sum, i) => sum + i.amount_in_cents, 0);
      const proportion = item.amount_in_cents / totalAmount;
      return Math.round(item.amount_in_cents - (discount.discount_value * proportion));
    }
    return item.amount_in_cents;
  }, [discount, lineItems]);

  const totals = useMemo(() => {
    const subtotal = lineItems.reduce((sum, item) => 
      sum + (item.amount_in_cents * item.quantity), 0
    );
    
    const discountAmount = lineItems.reduce((sum, item) => 
      sum + ((item.amount_in_cents - calculateDiscountedPrice(item)) * item.quantity), 0
    );
    
    const total = subtotal - discountAmount;
    
    return { subtotal, discountAmount, total };
  }, [lineItems, calculateDiscountedPrice]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pricing Summary</CardTitle>
        {trial && (
          <Badge variant="success" className="w-fit">
            {trial.trial_days} day free trial
          </Badge>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {showBreakdown && (
          <>
            <div className="space-y-3">
              {lineItems.map((item) => (
                <div key={item.id} className="flex justify-between items-start">
                  <div className="flex-1">
                    <p className="font-medium">{item.description}</p>
                    <p className="text-sm text-muted-foreground">
                      {item.quantity} × {formatCurrency(item.unit_amount_in_cents)}
                    </p>
                  </div>
                  <div className="text-right">
                    {discount && item.amount_in_cents !== calculateDiscountedPrice(item) && (
                      <p className="text-sm line-through text-muted-foreground">
                        {formatCurrency(item.amount_in_cents)}
                      </p>
                    )}
                    <p className="font-medium">
                      {formatCurrency(calculateDiscountedPrice(item))}
                    </p>
                  </div>
                </div>
              ))}
            </div>
            <Separator />
          </>
        )}
        
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Subtotal</span>
            <span>{formatCurrency(totals.subtotal)}</span>
          </div>
          
          {discount && totals.discountAmount > 0 && (
            <div className="flex justify-between text-sm text-green-600">
              <span>
                Discount
                {discount.code && ` (${discount.code})`}
              </span>
              <span>-{formatCurrency(totals.discountAmount)}</span>
            </div>
          )}
          
          <Separator />
          
          <div className="flex justify-between font-bold">
            <span>Total</span>
            <span className="text-lg">{formatCurrency(totals.total)}</span>
          </div>
          
          {subscription?.interval && (
            <p className="text-sm text-muted-foreground text-center">
              Billed {subscription.interval}
            </p>
          )}
          
          {trial && (
            <Alert>
              <AlertDescription className="text-sm">
                Your trial starts today. You won't be charged until {formatDate(trial.trial_end_date)}.
              </AlertDescription>
            </Alert>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
```

### 4. Page Implementations

#### Enhanced Dashboard Page

```typescript
// app/merchants/dashboard/page.tsx
export default function DashboardPage() {
  useRealtimeMetrics(); // WebSocket connection for live updates
  
  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
        <div className="flex items-center gap-2">
          <PeriodSelector />
          <CurrencySelector />
          <ExportButton />
        </div>
      </div>
      
      <Suspense fallback={<DashboardSkeleton />}>
        <MetricsDashboard />
      </Suspense>
      
      <Tabs defaultValue="revenue" className="space-y-4">
        <TabsList>
          <TabsTrigger value="revenue">Revenue</TabsTrigger>
          <TabsTrigger value="customers">Customers</TabsTrigger>
          <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
          <TabsTrigger value="gas">Gas Analytics</TabsTrigger>
        </TabsList>
        
        <TabsContent value="revenue">
          <RevenueAnalytics />
        </TabsContent>
        <TabsContent value="customers">
          <CustomerAnalytics />
        </TabsContent>
        <TabsContent value="subscriptions">
          <SubscriptionAnalytics />
        </TabsContent>
        <TabsContent value="gas">
          <GasAnalytics />
        </TabsContent>
      </Tabs>
    </div>
  );
}
```

#### B2B Payment Flow Pages

```typescript
// app/merchants/invoices/[id]/request-payment.tsx
export function PaymentRequestModal({ 
  invoice, 
  isOpen,
  onClose 
}: PaymentRequestModalProps) {
  const [isLoading, setIsLoading] = useState(false);
  const { register, handleSubmit, formState: { errors } } = useForm<PaymentRequestForm>();

  const onSubmit = async (data: PaymentRequestForm) => {
    setIsLoading(true);
    try {
      const response = await api.invoices.requestPayment(invoice.id, {
        customer_email: data.customer_email,
        delegation_amount: data.delegation_amount,
        expires_in_hours: data.expires_in_hours,
        message: data.message,
      });
      
      toast.success('Payment request sent successfully');
      onClose(response.payment_link);
    } catch (error) {
      toast.error('Failed to send payment request');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Request Payment</DialogTitle>
          <DialogDescription>
            Send a delegation-based payment request for invoice #{invoice.invoice_number}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <Label htmlFor="customer_email">Customer Email</Label>
            <Input 
              id="customer_email"
              type="email" 
              {...register('customer_email', { 
                required: 'Email is required',
                pattern: {
                  value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                  message: 'Invalid email address'
                }
              })}
              placeholder="customer@example.com"
            />
            {errors.customer_email && (
              <p className="text-sm text-red-500 mt-1">{errors.customer_email.message}</p>
            )}
          </div>

          <div>
            <Label htmlFor="delegation_amount">Delegation Amount</Label>
            <div className="flex gap-2">
              <Input 
                id="delegation_amount"
                type="number"
                step="0.01"
                {...register('delegation_amount', { 
                  required: 'Amount is required',
                  min: { value: invoice.amount_due / 100, message: 'Must cover invoice amount' }
                })}
                placeholder="1000"
              />
              <Select defaultValue="USDC">
                <SelectTrigger className="w-24">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="USDC">USDC</SelectItem>
                  <SelectItem value="USDT">USDT</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <p className="text-sm text-muted-foreground mt-1">
              Maximum amount the customer can delegate for payment
            </p>
            {errors.delegation_amount && (
              <p className="text-sm text-red-500 mt-1">{errors.delegation_amount.message}</p>
            )}
          </div>

          <div>
            <Label htmlFor="expires_in_hours">Expires In</Label>
            <Select {...register('expires_in_hours', { valueAsNumber: true })} defaultValue="24">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1">1 hour</SelectItem>
                <SelectItem value="6">6 hours</SelectItem>
                <SelectItem value="24">24 hours</SelectItem>
                <SelectItem value="72">3 days</SelectItem>
                <SelectItem value="168">7 days</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label htmlFor="message">Message (Optional)</Label>
            <Textarea 
              id="message"
              {...register('message')}
              placeholder="Add a note for your customer..."
              rows={3}
            />
          </div>

          <div className="bg-muted p-4 rounded-lg space-y-2">
            <div className="flex justify-between text-sm">
              <span>Invoice Amount</span>
              <span className="font-medium">{formatCurrency(invoice.amount_due)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span>Customer</span>
              <span className="font-medium">{invoice.customer.name}</span>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onClose()}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Sending...
                </>
              ) : (
                'Send Payment Request'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// app/pay/[code]/page.tsx
export default function PaymentLinkPage({ params }: { params: { code: string } }) {
  const { code } = params;
  const [paymentLink, setPaymentLink] = useState<PaymentLink | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isProcessing, setIsProcessing] = useState(false);
  const [paymentComplete, setPaymentComplete] = useState(false);

  useEffect(() => {
    api.paymentLinks.getPublic(code)
      .then(setPaymentLink)
      .catch(() => toast.error('Invalid payment link'))
      .finally(() => setIsLoading(false));
  }, [code]);

  const handlePayment = async () => {
    if (!paymentLink) return;
    
    setIsProcessing(true);
    try {
      // Initialize delegation with MetaMask
      const delegation = await initializeDelegation({
        recipient: paymentLink.recipient_address,
        amount: paymentLink.amount,
        token: paymentLink.token_address,
        network: paymentLink.network,
      });

      // Submit delegation transaction
      const tx = await submitDelegation(delegation);
      
      // Confirm payment with backend
      await api.paymentLinks.confirmPayment(paymentLink.id, {
        transaction_hash: tx.hash,
        delegation_id: delegation.id,
      });
      
      setPaymentComplete(true);
      toast.success('Payment successful!');
    } catch (error) {
      console.error('Payment failed:', error);
      toast.error(error.message || 'Payment failed');
    } finally {
      setIsProcessing(false);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (!paymentLink) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardContent className="pt-6">
            <div className="text-center">
              <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h2 className="text-xl font-semibold mb-2">Invalid Payment Link</h2>
              <p className="text-muted-foreground">
                This payment link is invalid or has expired.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (paymentComplete) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <Card className="w-full max-w-md">
          <CardContent className="pt-6">
            <div className="text-center">
              <CheckCircle className="h-16 w-16 text-green-500 mx-auto mb-4" />
              <h2 className="text-2xl font-bold mb-2">Payment Successful!</h2>
              <p className="text-muted-foreground mb-6">
                Your payment has been processed successfully.
              </p>
              <Button asChild>
                <Link href={paymentLink.success_url || '/'}>
                  Continue
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Payment Request</CardTitle>
          <CardDescription>
            from {paymentLink.merchant_name}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* QR Code for mobile sharing */}
          <div className="flex justify-center p-4 bg-white rounded-lg">
            <QRCodeSVG 
              value={window.location.href} 
              size={200}
              level="H"
              includeMargin
            />
          </div>

          {/* Payment Details */}
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Amount</span>
              <span className="font-bold text-lg">
                {formatCurrency(paymentLink.amount)}
              </span>
            </div>
            
            {paymentLink.invoice && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-muted-foreground">Invoice</span>
                <span className="font-medium">
                  #{paymentLink.invoice.number}
                </span>
              </div>
            )}
            
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Network</span>
              <Badge variant="secondary">
                {paymentLink.network_name}
              </Badge>
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Expires</span>
              <span className="text-sm">
                {formatRelativeTime(paymentLink.expires_at)}
              </span>
            </div>
            
            {paymentLink.message && (
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-sm">{paymentLink.message}</p>
              </div>
            )}
          </div>

          {/* Gas Fee Notice */}
          {paymentLink.gas_sponsored && (
            <Alert className="bg-green-50 border-green-200">
              <Fuel className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-800">
                Gas fees will be covered by the merchant
              </AlertDescription>
            </Alert>
          )}

          {/* Payment Button */}
          <Button 
            className="w-full" 
            size="lg"
            onClick={handlePayment}
            disabled={isProcessing}
          >
            {isProcessing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Processing Payment...
              </>
            ) : (
              <>
                <Wallet className="mr-2 h-4 w-4" />
                Pay with MetaMask
              </>
            )}
          </Button>

          <p className="text-xs text-center text-muted-foreground">
            Powered by MetaMask Delegation Toolkit
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
```

#### Invoice Management Pages

```typescript
// app/merchants/invoices/page.tsx
export default function InvoicesPage() {
  const { viewMode } = useInvoiceUI();
  
  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Invoices</h2>
        <div className="flex items-center gap-2">
          <ViewModeToggle />
          <FilterPopover />
          <Link href="/merchants/invoices/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Invoice
            </Button>
          </Link>
        </div>
      </div>
      
      <InvoiceStats />
      
      {viewMode === 'table' ? (
        <InvoiceTable />
      ) : (
        <InvoiceCards />
      )}
    </div>
  );
}

// app/merchants/invoices/[id]/page.tsx
export default function InvoiceDetailPage({ params }: { params: { id: string } }) {
  const { data: invoice, isLoading } = useInvoice(params.id);
  
  if (isLoading) return <InvoiceDetailSkeleton />;
  if (!invoice) return <NotFound />;
  
  return (
    <div className="flex-1 space-y-6 p-8 pt-6">
      <InvoiceHeader invoice={invoice} />
      
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <InvoiceLineItems items={invoice.line_items} />
          <InvoicePayments invoiceId={invoice.id} />
          <InvoiceActivity invoice={invoice} />
        </div>
        
        <div className="space-y-6">
          <InvoiceSummary invoice={invoice} />
          <InvoiceActions invoice={invoice} />
          <CustomerCard customer={invoice.customer} />
        </div>
      </div>
    </div>
  );
}
```

### 5. Mobile Responsiveness

All components must be fully responsive:

```typescript
// components/dashboard/ResponsiveMetrics.tsx
export function ResponsiveMetrics({ metrics }) {
  return (
    <>
      {/* Desktop: 4 columns */}
      <div className="hidden lg:grid lg:grid-cols-4 gap-4">
        {metrics.map(metric => <MetricCard key={metric.id} {...metric} />)}
      </div>
      
      {/* Tablet: 2 columns */}
      <div className="hidden md:grid lg:hidden md:grid-cols-2 gap-4">
        {metrics.map(metric => <MetricCard key={metric.id} {...metric} />)}
      </div>
      
      {/* Mobile: Carousel */}
      <div className="md:hidden">
        <MetricsCarousel metrics={metrics} />
      </div>
    </>
  );
}
```

### 6. Performance Optimizations

#### Code Splitting
```typescript
// Lazy load heavy components
const InvoiceBuilder = lazy(() => import('./components/invoices/InvoiceBuilder'));
const RevenueChart = lazy(() => import('./components/charts/RevenueChart'));
const ExportModal = lazy(() => import('./components/modals/ExportModal'));
```

#### Data Caching Strategy
```typescript
// hooks/data/useCachedCurrency.ts
export function useCachedCurrency() {
  // Cache currency data in localStorage with TTL
  const [cachedData, setCachedData] = useLocalStorage('currency-data', null);
  
  const { data: freshData } = useQuery({
    queryKey: ['currencies'],
    queryFn: fetchCurrencies,
    staleTime: 24 * 60 * 60 * 1000, // 24 hours
    initialData: cachedData,
  });
  
  useEffect(() => {
    if (freshData) {
      setCachedData({
        data: freshData,
        timestamp: Date.now(),
      });
    }
  }, [freshData]);
  
  return freshData;
}
```

### 7. Accessibility

All components must meet WCAG 2.1 AA standards:

```typescript
// components/ui/accessible-metric-card.tsx
export function AccessibleMetricCard({ title, value, change, trend }) {
  return (
    <Card role="article" aria-label={`${title} metric`}>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          <span id={`metric-${title}-label`}>{title}</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div 
          className="text-2xl font-bold"
          aria-labelledby={`metric-${title}-label`}
          aria-describedby={`metric-${title}-change`}
        >
          {value}
        </div>
        {change && (
          <p 
            id={`metric-${title}-change`}
            className="text-xs text-muted-foreground"
            aria-label={`Change: ${change > 0 ? 'up' : 'down'} ${Math.abs(change)}%`}
          >
            <TrendIcon className="inline h-4 w-4" aria-hidden="true" />
            <span className="sr-only">{trend}</span>
            {change}%
          </p>
        )}
      </CardContent>
    </Card>
  );
}
```

## Currency Migration Strategy

### Remove Hardcoded Currency References
All hardcoded currency symbols and formatting must be replaced with database-driven approach:

1. **Replace Constants File**
   ```typescript
   // OLD: lib/constants/currency.ts
   export const CURRENCY_SYMBOLS = { USD: '$', EUR: '€' };
   
   // NEW: Use useCurrency hook everywhere
   const { formatCurrency, getCurrencyData } = useCurrency();
   ```

2. **Update All Components**
   - Search and replace all instances of hardcoded `$` symbols
   - Replace manual formatting like `${amount} USD`
   - Use formatCurrency function consistently
   - Update currency selectors to use getSupportedCurrencies()

3. **Migration Checklist**
   - [ ] Remove `/lib/constants/currency.ts`
   - [ ] Update all product components
   - [ ] Update all invoice components
   - [ ] Update dashboard displays
   - [ ] Update payment components
   - [ ] Update subscription displays
   - [ ] Update all price input fields
   - [ ] Update export/report generation

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
1. **Currency System**
   - Implement currency hooks and formatting
   - Replace all hardcoded USD references
   - Add currency selector component
   - Test with multiple currencies

2. **Enhanced API Integration**
   - Extend API client with new endpoints
   - Create React Query hooks for all features
   - Implement WebSocket connection
   - Add optimistic updates

3. **State Management**
   - Create new Zustand stores
   - Migrate existing state patterns
   - Add persistence where needed

### Phase 2: Dashboard & Analytics (Week 3-4)
1. **Dashboard Replacement**
   - Replace static dashboard with real data
   - Implement all metric cards
   - Add revenue charts
   - Create period selector

2. **Analytics Pages**
   - Revenue analytics with charts
   - Customer analytics
   - Subscription metrics
   - Gas fee analytics

3. **Real-time Updates**
   - WebSocket integration
   - Live metric updates
   - Payment notifications

### Phase 3: Invoice System (Week 5-6)
1. **Invoice Management**
   - Invoice list with filters
   - Invoice detail page
   - Invoice status management
   - Payment tracking

2. **Invoice Builder**
   - Multi-step form
   - Line item management
   - Tax calculation UI
   - Preview system

3. **Customer Invoice Portal**
   - Public invoice view
   - Payment interface
   - Download/print options

### Phase 4: Financial Features (Week 7-8)
1. **Payment Management**
   - Payment list and details
   - Gas fee breakdown
   - Refund interface
   - Payment links

2. **Tax Management**
   - Customer tax profiles
   - Tax exemption UI
   - Jurisdiction display
   - Tax reports

3. **Gas Sponsorship**
   - Settings interface
   - Analytics dashboard
   - Rule configuration
   - Budget tracking

### Phase 5: Polish & Optimization (Week 9-10)
1. **Performance**
   - Implement code splitting
   - Optimize bundle size
   - Add caching strategies
   - Performance monitoring

2. **Mobile Experience**
   - Responsive testing
   - Touch optimizations
   - Mobile-specific features

3. **Accessibility**
   - WCAG compliance
   - Keyboard navigation
   - Screen reader testing
   - High contrast mode

## Testing Strategy

### Unit Testing
```typescript
// __tests__/components/CurrencyDisplay.test.tsx
describe('CurrencyDisplay', () => {
  it('formats currency according to fiat_currencies data', () => {
    const currency = {
      code: 'EUR',
      symbol: '€',
      decimal_places: 2,
      symbol_position: 'before',
    };
    
    const { getByText } = render(
      <CurrencyDisplay 
        amountInCents={12350} 
        currency={currency}
      />
    );
    
    expect(getByText('€123.50')).toBeInTheDocument();
  });
});
```

### Integration Testing
```typescript
// __tests__/integration/invoice-flow.test.tsx
describe('Invoice Creation Flow', () => {
  it('creates invoice with tax calculation', async () => {
    const { user } = renderWithProviders(<InvoiceBuilder />);
    
    // Select customer
    await user.click(screen.getByText('Select Customer'));
    await user.click(screen.getByText('Acme Corp'));
    
    // Add line items
    await user.click(screen.getByText('Add Product'));
    await user.type(screen.getByLabelText('Description'), 'Subscription');
    await user.type(screen.getByLabelText('Amount'), '100');
    
    // Verify tax calculation
    await waitFor(() => {
      expect(screen.getByText('Tax: $10.00')).toBeInTheDocument();
    });
    
    // Submit
    await user.click(screen.getByText('Create Invoice'));
    
    expect(mockApi.invoices.create).toHaveBeenCalledWith(
      expect.objectContaining({
        customer_id: 'acme-corp-id',
        line_items: expect.arrayContaining([
          expect.objectContaining({
            description: 'Subscription',
            amount_in_cents: 10000,
          })
        ]),
        tax_calculations: expect.any(Array),
      })
    );
  });
});
```

### E2E Testing
```typescript
// e2e/invoice-payment-flow.spec.ts
test('complete invoice payment flow', async ({ page }) => {
  // Create invoice
  await page.goto('/merchants/invoices/new');
  await page.selectOption('[name="customer"]', 'customer-123');
  await page.fill('[name="line_items[0].description"]', 'API Access');
  await page.fill('[name="line_items[0].amount"]', '99.99');
  await page.click('button:has-text("Create Invoice")');
  
  // Send to customer
  await page.click('button:has-text("Send Invoice")');
  
  // Customer receives and pays
  const invoiceUrl = await page.getAttribute('[data-testid="invoice-link"]', 'href');
  await page.goto(invoiceUrl);
  
  // Verify invoice display
  await expect(page.locator('text=API Access')).toBeVisible();
  await expect(page.locator('text=$99.99')).toBeVisible();
  
  // Pay invoice
  await page.click('button:has-text("Pay Now")');
  
  // Verify payment confirmation
  await expect(page.locator('text=Payment Successful')).toBeVisible();
});
```

## Migration Strategy

### 1. Gradual Component Migration
- Start with leaf components (buttons, cards)
- Move up to feature components
- Finally migrate pages
- Keep old components during transition

### 2. Feature Flags
```typescript
// lib/feature-flags.ts
export const features = {
  newDashboard: process.env.NEXT_PUBLIC_NEW_DASHBOARD === 'true',
  invoiceSystem: process.env.NEXT_PUBLIC_INVOICE_SYSTEM === 'true',
  gasSponsorship: process.env.NEXT_PUBLIC_GAS_SPONSORSHIP === 'true',
};

// Usage
{features.newDashboard ? <NewDashboard /> : <OldDashboard />}
```

### 3. Data Migration
- Ensure backward compatibility
- Provide data transformation utilities
- Handle missing fields gracefully

## Success Metrics

### Performance Metrics
- Initial page load < 3s
- Time to interactive < 5s
- Lighthouse score > 90
- Bundle size < 500KB

### User Experience Metrics
- Dashboard data load time < 2s
- Invoice creation time < 3 minutes
- Payment completion rate > 85%
- Error rate < 0.1%

### Business Metrics
- Increased invoice payment rate
- Reduced support tickets
- Higher user engagement
- Improved conversion rates

## Conclusion

This frontend implementation plan provides a comprehensive roadmap for transforming the Cyphera web application into a powerful financial management platform. By leveraging modern React patterns, real-time updates, and thoughtful UX design, we'll create an interface that makes complex financial operations simple and intuitive for users.

The phased approach ensures we can deliver value incrementally while maintaining system stability. Each phase builds upon the previous work, ultimately resulting in a cohesive, performant, and user-friendly application that fully utilizes the enhanced backend capabilities.