import React from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { 
  BarChart3, 
  TrendingUp, 
  Users, 
  CreditCard, 
  RefreshCw,
  Rocket,
  ArrowRight,
  FileText,
  Settings
} from 'lucide-react';
import Link from 'next/link';

interface EmptyStateProps {
  type: 'no-data' | 'calculating' | 'error';
  onRefresh?: () => void;
  isRefreshing?: boolean;
  isFetching?: boolean;
}

export function DashboardEmptyState({ type, onRefresh, isRefreshing, isFetching }: EmptyStateProps) {
  if (type === 'calculating') {
    return (
      <Card className="border-dashed">
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <div className="rounded-full bg-primary/10 p-4 mb-4">
            <RefreshCw className="h-8 w-8 text-primary animate-spin" />
          </div>
          <h3 className="text-lg font-semibold mb-2">Calculating Your Metrics</h3>
          <p className="text-muted-foreground max-w-md mb-4">
            We're processing your data to generate analytics. This usually takes a few moments.
          </p>
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground justify-center">
              <div className="h-2 w-2 bg-primary rounded-full animate-pulse" />
              {isFetching ? 'Checking for updates...' : 'Analytics will refresh automatically'}
            </div>
            <div className="text-xs text-muted-foreground">
              Auto-refreshing every 5 seconds
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (type === 'error') {
    return (
      <Card className="border-dashed border-destructive/50">
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <div className="rounded-full bg-destructive/10 p-4 mb-4">
            <BarChart3 className="h-8 w-8 text-destructive" />
          </div>
          <h3 className="text-lg font-semibold mb-2">Unable to Load Analytics</h3>
          <p className="text-muted-foreground max-w-md mb-4">
            There was an error loading your analytics data. Please try refreshing.
          </p>
          <Button 
            onClick={onRefresh} 
            disabled={isRefreshing}
            variant="outline"
          >
            {isRefreshing ? (
              <>
                <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                Refreshing...
              </>
            ) : (
              <>
                <RefreshCw className="mr-2 h-4 w-4" />
                Retry
              </>
            )}
          </Button>
        </CardContent>
      </Card>
    );
  }

  // Default: no-data state
  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center justify-center py-16 text-center">
        <div className="rounded-full bg-muted p-4 mb-4">
          <Rocket className="h-8 w-8 text-muted-foreground" />
        </div>
        <h3 className="text-lg font-semibold mb-2">Welcome to Your Dashboard!</h3>
        <p className="text-muted-foreground max-w-md mb-6">
          Start accepting crypto payments and subscriptions to see your analytics here.
        </p>
        <div className="grid gap-4 w-full max-w-md">
          <Link href="/merchants/products" className="w-full">
            <Button className="w-full" variant="default">
              <CreditCard className="mr-2 h-4 w-4" />
              Create Your First Product
            </Button>
          </Link>
          <Link href="/merchants/settings" className="w-full">
            <Button className="w-full" variant="outline">
              <Settings className="mr-2 h-4 w-4" />
              Generate API Keys
            </Button>
          </Link>
          <Link href="/docs/getting-started" className="w-full">
            <Button className="w-full" variant="ghost">
              <FileText className="mr-2 h-4 w-4" />
              View Integration Guide
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}

interface ChartEmptyStateProps {
  title: string;
  description: string;
  icon?: React.ReactNode;
  isCalculating?: boolean;
}

export function ChartEmptyState({ 
  title, 
  description, 
  icon = <BarChart3 className="h-6 w-6" />,
  isCalculating = false 
}: ChartEmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center h-64 text-center px-4">
      <div className={`rounded-full p-3 mb-3 ${isCalculating ? 'bg-primary/10' : 'bg-muted'}`}>
        {isCalculating ? (
          <RefreshCw className="h-6 w-6 text-primary animate-spin" />
        ) : (
          <div className="text-muted-foreground">{icon}</div>
        )}
      </div>
      <h4 className="font-medium mb-1">{title}</h4>
      <p className="text-sm text-muted-foreground max-w-xs">
        {isCalculating ? 'Calculating metrics...' : description}
      </p>
    </div>
  );
}

interface MetricCardEmptyStateProps {
  isCalculating?: boolean;
}

export function MetricCardEmptyState({ isCalculating }: MetricCardEmptyStateProps) {
  if (isCalculating) {
    return (
      <div className="space-y-2">
        <div className="h-8 w-24 bg-muted animate-pulse rounded" />
        <div className="h-4 w-16 bg-muted animate-pulse rounded" />
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="text-2xl font-bold">--</div>
      <div className="text-sm text-muted-foreground">No data yet</div>
    </div>
  );
}

interface OnboardingBannerProps {
  currentStep: number;
  totalSteps: number;
  onDismiss?: () => void;
}

export function OnboardingBanner({ currentStep, totalSteps, onDismiss }: OnboardingBannerProps) {
  const steps = [
    { label: 'Create Product', href: '/merchants/products' },
    { label: 'Generate API Keys', href: '/merchants/settings' },
    { label: 'Integrate SDK', href: '/docs/sdk-integration' },
    { label: 'Accept First Payment', href: '/merchants/transactions' },
  ];

  return (
    <Card className="bg-primary/5 border-primary/20">
      <CardContent className="py-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Rocket className="h-5 w-5 text-primary" />
            <h3 className="font-semibold">Get Started with Cyphera</h3>
          </div>
          {onDismiss && (
            <Button variant="ghost" size="sm" onClick={onDismiss}>
              Dismiss
            </Button>
          )}
        </div>
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <div className="flex-1 bg-muted rounded-full h-2">
              <div 
                className="bg-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${(currentStep / totalSteps) * 100}%` }}
              />
            </div>
            <span className="text-sm text-muted-foreground">
              {currentStep} of {totalSteps}
            </span>
          </div>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-2">
            {steps.map((step, index) => (
              <Link 
                key={index} 
                href={step.href}
                className={`text-sm px-3 py-2 rounded-md text-center transition-colors ${
                  index < currentStep 
                    ? 'bg-primary/10 text-primary' 
                    : index === currentStep 
                    ? 'bg-primary text-primary-foreground' 
                    : 'bg-muted text-muted-foreground hover:bg-muted/80'
                }`}
              >
                {step.label}
              </Link>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}