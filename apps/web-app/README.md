# Web Application

> **Navigation:** [← Root README](../../README.md) | [Main API →](../api/README.md) | [Architecture →](../../docs/architecture.md)

The Cyphera web application is a Next.js 15 frontend that provides merchant dashboards and customer portals for Web3 subscription management.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Development](#development)
- [Authentication & Web3](#authentication--web3)
- [Component Library](#component-library)
- [State Management](#state-management)
- [API Integration](#api-integration)
- [Testing](#testing)
- [Deployment](#deployment)

## Overview

The web application serves as the primary interface for both merchants and customers, built with modern React patterns and Web3 integration.

### Key Features
- **Next.js 15** with App Router and React Server Components
- **Web3Auth Integration** for social login with Web3 wallet creation
- **Merchant Dashboard** for subscription and customer management
- **Customer Portal** for subscription payments and wallet management
- **MetaMask Delegation** for automatic subscription billing
- **Circle Wallets** integration for programmable wallets
- **Responsive Design** with Tailwind CSS and shadcn/ui components
- **Real-time Updates** with optimistic UI patterns

### User Flows

#### Merchant Flow
1. **Authentication** - Sign in via Web3Auth (Google, Discord, etc.)
2. **Onboarding** - Set up workspace and payment configuration
3. **Product Creation** - Define subscription plans and pricing
4. **Customer Management** - View and manage customer subscriptions
5. **Analytics** - Monitor subscription metrics and payments

#### Customer Flow
1. **Discovery** - Browse available subscription products
2. **Authentication** - Connect wallet or create embedded wallet
3. **Subscription** - Select plan and set up delegation for auto-payments
4. **Management** - View active subscriptions and payment history
5. **Wallet Operations** - Manage funds and delegation permissions

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Merchant      │    │   Customer      │    │   Public        │
│   Dashboard     │    │   Portal        │    │   Pages         │
│   /merchants    │    │   /customers    │    │   /public       │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   Next.js App Router    │
                    │   (Route Handlers)      │
                    └────────────┬────────────┘
                                 │
            ┌────────────────────┼────────────────────┐
            │                    │                    │
   ┌────────▼─────────┐ ┌────────▼─────────┐ ┌───────▼────────┐
   │   Components     │ │   State Mgmt     │ │   API Client   │
   │   (UI Library)   │ │   (Zustand)      │ │   (Axios)      │
   └────────┬─────────┘ └──────────────────┘ └───────┬────────┘
            │                                        │
   ┌────────▼─────────┐                     ┌───────▼────────┐
   │   Web3 Layer     │                     │   Backend      │
   │ (Viem, Wagmi,    │                     │   API          │
   │  Web3Auth)       │                     │   (Go/Gin)     │
   └──────────────────┘                     └────────────────┘
```

### Directory Structure

```
apps/web-app/
├── src/
│   ├── app/                    # Next.js App Router
│   │   ├── (auth)/            # Auth layout group
│   │   ├── merchants/         # Merchant dashboard
│   │   ├── customers/         # Customer portal
│   │   ├── public/            # Public pages
│   │   ├── api/               # API route handlers
│   │   └── globals.css        # Global styles
│   ├── components/            # React components
│   │   ├── ui/               # Base UI components (shadcn/ui)
│   │   ├── auth/             # Authentication components
│   │   ├── products/         # Product management
│   │   ├── customers/        # Customer components
│   │   ├── wallets/          # Wallet management
│   │   └── providers/        # Context providers
│   ├── hooks/                # Custom React hooks
│   │   ├── auth/            # Authentication hooks
│   │   ├── web3/            # Web3 interaction hooks
│   │   └── data/            # Data fetching hooks
│   ├── lib/                 # Utility libraries
│   │   ├── api/             # API client configuration
│   │   ├── auth/            # Authentication utilities
│   │   ├── web3/            # Web3 configuration
│   │   └── utils.ts         # General utilities
│   ├── store/               # State management
│   │   ├── auth.ts          # Authentication state
│   │   ├── wallet.ts        # Wallet state
│   │   └── network.ts       # Network state
│   └── types/               # TypeScript type definitions
├── docs/                    # Documentation
│   ├── cyphera-implementation-plan.md
│   └── components.md
├── public/                  # Static assets
├── package.json
├── tailwind.config.ts
├── next.config.js
└── README.md               # This file
```

## Development

### Prerequisites
- Node.js 18 or later
- npm 9 or later
- Environment variables configured

### Installation
```bash
# From project root
npm run install:ts

# Or directly in web-app
cd apps/web-app
npm install --legacy-peer-deps
```

### Running Locally

#### Development Server
```bash
# From project root
npm run dev:web

# Or directly
cd apps/web-app
npm run dev
```

The application will start on `http://localhost:3000`.

#### Environment Variables
Create `.env.local` file:
```bash
cd apps/web-app
cp .env.example .env.local
```

Configure the following variables:
```bash
# Web3Auth Configuration
NEXT_PUBLIC_WEB3AUTH_CLIENT_ID="your_web3auth_client_id"
NEXT_PUBLIC_WEB3AUTH_NETWORK="sapphire_devnet" # or sapphire_mainnet

# Pimlico Configuration (Required for Account Abstraction)
NEXT_PUBLIC_PIMLICO_API_KEY="your_pimlico_api_key" # Get from https://pimlico.io

# API Endpoints
NEXT_PUBLIC_API_URL="http://localhost:8080"
NEXT_PUBLIC_DELEGATION_SERVER_URL="http://localhost:50051"

# Circle API
NEXT_PUBLIC_CIRCLE_APP_ID="your_circle_app_id"

# Blockchain Networks (RPC URLs)
NEXT_PUBLIC_INFURA_API_KEY="your_infura_api_key" # Used for RPC endpoints
NEXT_PUBLIC_ETHEREUM_RPC_URL="https://eth-sepolia.g.alchemy.com/v2/your_key"
NEXT_PUBLIC_POLYGON_RPC_URL="https://polygon-mumbai.g.alchemy.com/v2/your_key"

# Development Settings
NEXT_PUBLIC_NODE_ENV="development"
NEXT_PUBLIC_LOG_LEVEL="debug"
```

### Development Commands
```bash
# Development server
npm run dev

# Build application
npm run build

# Start production server
npm run start

# Run tests
npm run test

# Lint code
npm run lint

# Type checking
npm run type-check

# Format code
npm run format
```

## Authentication & Web3

### Web3Auth Integration
The application uses Web3Auth for user authentication with embedded wallet creation:

```typescript
// Web3Auth configuration
const web3AuthConfig = {
  clientId: process.env.NEXT_PUBLIC_WEB3AUTH_CLIENT_ID,
  web3AuthNetwork: WEB3AUTH_NETWORK.SAPPHIRE_DEVNET,
  chainConfig: {
    chainNamespace: CHAIN_NAMESPACES.EIP155,
    chainId: "0xaa36a7", // Sepolia
    rpcTarget: process.env.NEXT_PUBLIC_ETHEREUM_RPC_URL,
  },
  uiConfig: {
    theme: "dark",
    loginMethodsOrder: ["google", "discord", "twitter"],
  }
};
```

### Authentication Flow
```typescript
// Login component example
const LoginButton = () => {
  const { login, user, loading } = useWeb3Auth();
  
  return (
    <button 
      onClick={() => login()}
      disabled={loading}
    >
      {loading ? 'Connecting...' : 'Login with Web3Auth'}
    </button>
  );
};
```

### Wallet Integration

#### Circle Programmable Wallets
```typescript
// Circle wallet integration
const useCircleWallet = () => {
  const createWallet = async (blockchain: string) => {
    const response = await fetch('/api/circle/wallets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ blockchain })
    });
    return response.json();
  };
  
  return { createWallet };
};
```

#### MetaMask Delegation
```typescript
// Delegation setup for subscriptions
const useDelegation = () => {
  const setupDelegation = async (productId: string) => {
    const delegation = await createDelegation({
      delegate: PLATFORM_DELEGATE_ADDRESS,
      authority: userWallet.address,
      caveats: [
        {
          type: 'allowedTargets',
          value: [USDC_CONTRACT_ADDRESS]
        },
        {
          type: 'spendingLimit',
          value: subscriptionAmount
        }
      ]
    });
    
    return delegation;
  };
  
  return { setupDelegation };
};
```

## Component Library

### UI Components (shadcn/ui)
The application uses shadcn/ui for consistent, accessible components:

```typescript
// Example component usage
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const SubscriptionCard = ({ subscription }) => (
  <Card>
    <CardHeader>
      <CardTitle>{subscription.product.name}</CardTitle>
      <Badge variant={subscription.status === 'active' ? 'default' : 'secondary'}>
        {subscription.status}
      </Badge>
    </CardHeader>
    <CardContent>
      <Button onClick={() => manageSubscription(subscription.id)}>
        Manage
      </Button>
    </CardContent>
  </Card>
);
```

### Custom Components

#### Authentication Components
- `<Web3AuthLogin />` - Web3Auth login interface
- `<WalletConnector />` - Wallet connection management
- `<RoleSwitcher />` - Switch between merchant/customer roles

#### Product Management
- `<ProductCreationDialog />` - Multi-step product creation
- `<PricingSelector />` - Price and token configuration
- `<NetworkTokenSelector />` - Blockchain network selection

#### Wallet Components
- `<WalletBalance />` - Display wallet balances
- `<DelegationButton />` - Set up subscription delegations
- `<TransactionHistory />` - Transaction list with filtering

#### Customer Management
- `<CustomerList />` - Paginated customer table
- `<CustomerDetails />` - Customer profile and subscription history
- `<SubscriptionMetrics />` - Analytics dashboard

### Component Patterns

#### Higher-Order Components (HOCs)
```typescript
// Authentication HOC
export const withAuth = <P extends object>(
  Component: React.ComponentType<P>
) => {
  return (props: P) => {
    const { user, loading } = useAuth();
    
    if (loading) return <LoadingSpinner />;
    if (!user) return <LoginPrompt />;
    
    return <Component {...props} />;
  };
};

// Usage
const ProtectedDashboard = withAuth(Dashboard);
```

#### Compound Components
```typescript
// Subscription management compound component
const SubscriptionManager = ({ children }) => (
  <div className="subscription-manager">{children}</div>
);

SubscriptionManager.Header = ({ title }) => (
  <header className="mb-6">{title}</header>
);

SubscriptionManager.List = ({ subscriptions }) => (
  <div className="space-y-4">
    {subscriptions.map(sub => <SubscriptionCard key={sub.id} {...sub} />)}
  </div>
);

// Usage
<SubscriptionManager>
  <SubscriptionManager.Header title="My Subscriptions" />
  <SubscriptionManager.List subscriptions={subscriptions} />
</SubscriptionManager>
```

## State Management

### Zustand Stores
The application uses Zustand for client-side state management:

#### Authentication Store
```typescript
// store/auth.ts
interface AuthState {
  user: User | null;
  workspace: Workspace | null;
  isLoading: boolean;
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  switchWorkspace: (workspaceId: string) => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  workspace: null,
  isLoading: false,
  
  login: async (credentials) => {
    set({ isLoading: true });
    try {
      const { user, workspace } = await authApi.login(credentials);
      set({ user, workspace, isLoading: false });
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },
  
  logout: () => {
    set({ user: null, workspace: null });
    authApi.logout();
  },
  
  switchWorkspace: async (workspaceId) => {
    const workspace = await workspaceApi.get(workspaceId);
    set({ workspace });
  }
}));
```

#### Wallet Store
```typescript
// store/wallet.ts
interface WalletState {
  wallets: Wallet[];
  selectedWallet: Wallet | null;
  balances: Record<string, TokenBalance[]>;
  fetchWallets: () => Promise<void>;
  selectWallet: (wallet: Wallet) => void;
  fetchBalances: (walletId: string) => Promise<void>;
}

export const useWalletStore = create<WalletState>((set, get) => ({
  wallets: [],
  selectedWallet: null,
  balances: {},
  
  fetchWallets: async () => {
    const wallets = await walletApi.list();
    set({ wallets });
  },
  
  selectWallet: (wallet) => {
    set({ selectedWallet: wallet });
  },
  
  fetchBalances: async (walletId) => {
    const balances = await walletApi.getBalances(walletId);
    set(state => ({
      balances: { ...state.balances, [walletId]: balances }
    }));
  }
}));
```

### Server State Management
For server state, the application uses TanStack Query (React Query):

```typescript
// hooks/data/use-products.ts
export const useProducts = () => {
  return useQuery({
    queryKey: ['products'],
    queryFn: productApi.list,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};

export const useCreateProduct = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: productApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries(['products']);
    },
  });
};
```

## API Integration

### API Client Configuration
```typescript
// lib/api/api-instance.ts
const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL,
  timeout: 30000,
});

// Request interceptor for authentication
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  const workspaceId = getCurrentWorkspaceId();
  
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  
  if (workspaceId) {
    config.headers['X-Workspace-ID'] = workspaceId;
  }
  
  return config;
});

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle authentication error
      authStore.logout();
      router.push('/login');
    }
    return Promise.reject(error);
  }
);
```

### API Service Layer
```typescript
// services/product-api.ts
export const productApi = {
  list: (): Promise<Product[]> =>
    apiClient.get('/products').then(res => res.data.data),
  
  get: (id: string): Promise<Product> =>
    apiClient.get(`/products/${id}`).then(res => res.data),
  
  create: (product: CreateProductRequest): Promise<Product> =>
    apiClient.post('/products', product).then(res => res.data),
  
  update: (id: string, updates: UpdateProductRequest): Promise<Product> =>
    apiClient.put(`/products/${id}`, updates).then(res => res.data),
  
  delete: (id: string): Promise<void> =>
    apiClient.delete(`/products/${id}`)
};
```

## Testing

### Test Setup
The application uses Jest and React Testing Library:

```typescript
// __tests__/components/ProductCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { ProductCard } from '@/components/products/ProductCard';

const mockProduct = {
  id: 'prod_123',
  name: 'Premium Plan',
  description: 'Access to premium features',
  prices: [{ amount: 2999, currency: 'USD', interval: 'month' }]
};

describe('ProductCard', () => {
  it('renders product information correctly', () => {
    render(<ProductCard product={mockProduct} />);
    
    expect(screen.getByText('Premium Plan')).toBeInTheDocument();
    expect(screen.getByText('Access to premium features')).toBeInTheDocument();
    expect(screen.getByText('$29.99/month')).toBeInTheDocument();
  });
  
  it('handles subscription click', () => {
    const onSubscribe = jest.fn();
    render(<ProductCard product={mockProduct} onSubscribe={onSubscribe} />);
    
    fireEvent.click(screen.getByText('Subscribe'));
    expect(onSubscribe).toHaveBeenCalledWith(mockProduct.id);
  });
});
```

### Testing Commands
```bash
# Run all tests
npm run test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run E2E tests (if configured)
npm run test:e2e
```

## Deployment

### Build Configuration
```javascript
// next.config.js
const nextConfig = {
  experimental: {
    optimizePackageImports: true,
  },
  
  webpack: (config, { isServer }) => {
    // Web3 polyfills for browser
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false,
      };
    }
    
    return config;
  },
  
  // Environment variables validation
  env: {
    CUSTOM_KEY: process.env.CUSTOM_KEY,
  },
};
```

### Production Build
```bash
# Build for production
npm run build

# Analyze bundle size
ANALYZE=true npm run build

# Start production server
npm run start
```

### Deployment Platforms

#### Vercel (Recommended)
1. Connect GitHub repository
2. Configure environment variables
3. Deploy automatically on push

#### Docker Deployment
```dockerfile
FROM node:18-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM node:18-alpine AS builder
WORKDIR /app
COPY . .
COPY --from=deps /app/node_modules ./node_modules
RUN npm run build

FROM node:18-alpine AS runner
WORKDIR /app
ENV NODE_ENV production
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000
CMD ["node", "server.js"]
```

---

## Related Documentation

- **[Implementation Plan](docs/cyphera-implementation-plan.md)** - Detailed improvement roadmap
- **[Component Guide](docs/components.md)** - Component library documentation
- **[Architecture Guide](../../docs/architecture.md)** - System overview
- **[API Reference](../../docs/api-reference.md)** - Backend API documentation

## Need Help?

- **[Troubleshooting](../../docs/troubleshooting.md)** - Common issues
- **[Contributing](../../docs/contributing.md)** - Development workflow
- **Next.js Documentation** - Framework documentation
- **GitHub Issues** - Bug reports and feature requests

---

*Last updated: $(date '+%Y-%m-%d')*
*Application Version: 2.0.0*