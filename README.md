# Cyphera Platform

[![Build Status](https://github.com/your-org/cyphera-api/workflows/CI/badge.svg)](https://github.com/your-org/cyphera-api/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/cyphera-api)](https://goreportcard.com/report/github.com/your-org/cyphera-api)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Cyphera is a comprehensive Web3 payment infrastructure platform that enables merchants to accept cryptocurrency subscriptions with automatic billing through MetaMask delegation. Built as a modern monorepo with Go, Node.js, and Next.js services.

## ⚡ Quick Start

Get started in under 10 minutes:

```bash
# Clone and install
git clone https://github.com/your-org/cyphera-api.git
cd cyphera-api
npm run install:all

# Setup environment
cp .env.example .env
# Edit .env with your configuration

# Start all services
npm run dev:all
```

**🌐 Access Points:**
- **Web App:** http://localhost:3000
- **API Server:** http://localhost:8080  
- **Health Check:** http://localhost:8080/health

**📖 For detailed setup instructions:** [Quick Start Guide →](docs/quick-start.md)

## 🏗️ Architecture

Cyphera consists of multiple integrated microservices:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │     Main API    │    │  Delegation     │
│   (Next.js)     │ ── │   (Go/Gin)      │ ── │  Server         │
│   Port: 3000    │    │   Port: 8080    │    │  (Node.js/gRPC) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │     PostgreSQL          │
                    │     Database            │
                    └─────────────────────────┘
```

### Core Services

| Service | Technology | Purpose | Port |
|---------|------------|---------|------|
| **[Web App](apps/web-app/README.md)** | Next.js 15 | Frontend interface | 3000 |
| **[Main API](apps/api/README.md)** | Go + Gin | Core business logic | 8080 |
| **[Delegation Server](apps/delegation-server/README.md)** | Node.js + gRPC | Blockchain operations | 50051 |
| **[Subscription Processor](apps/subscription-processor/README.md)** | Go | Background billing | - |

**📖 For detailed architecture information:** [Architecture Guide →](docs/architecture.md)

## 🚀 Development Commands

### Essential Commands
```bash
# Development servers
npm run dev:all              # Run all services
npm run dev:web              # Web app only  
npm run dev:api              # API server only
npm run dev:delegation       # Delegation server only

# Installation & Setup
npm run install:all          # Install all dependencies
npm run generate:all         # Generate code (SQLC, gRPC)
npm run setup               # Full setup (install + generate)

# Testing & Quality
npm run test:all            # Run all tests
npm run lint                # Lint code
npm run format              # Format code
npm run typecheck           # TypeScript checking

# Building
npm run build:all           # Build all services
npm run build:web           # Build web app
npm run clean               # Clean build artifacts
```

### Database Operations
```bash
docker-compose up postgres  # Start PostgreSQL
npm run db:migrate          # Apply migrations  
npm run db:reset           # Reset database
make gen                   # Regenerate SQLC code
```

## 📚 Documentation

### Getting Started
- **[Quick Start](docs/quick-start.md)** - 10-minute setup guide
- **[Architecture Overview](docs/architecture.md)** - System design and components
- **[API Reference](docs/api-reference.md)** - Complete API documentation

### Service Documentation  
- **[Web Application](apps/web-app/README.md)** - Frontend development guide
- **[Main API Server](apps/api/README.md)** - Backend API development  
- **[Delegation Server](apps/delegation-server/README.md)** - Blockchain operations
- **[Subscription Processor](apps/subscription-processor/README.md)** - Background jobs

### Operations
- **[Deployment Guide](docs/deployment.md)** - Production deployment
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions
- **[Contributing Guide](docs/contributing.md)** - Development workflow

## 🛠️ Technology Stack

### Backend Services
- **Languages:** Go 1.21+, Node.js 18+
- **Frameworks:** Gin (REST), gRPC, Express
- **Database:** PostgreSQL with SQLC
- **Deployment:** AWS Lambda, ECS, Docker

### Frontend
- **Framework:** Next.js 15 with App Router
- **Styling:** Tailwind CSS + shadcn/ui
- **State:** Zustand
- **Authentication:** Web3Auth + JWT

### Blockchain
- **Integration:** MetaMask Delegation Toolkit
- **Networks:** Ethereum, Polygon, Arbitrum
- **Libraries:** Viem, Wagmi
- **Wallets:** Circle Programmable Wallets

### Infrastructure
- **Cloud:** AWS (Lambda, ECS, RDS, Secrets Manager)
- **CI/CD:** GitHub Actions
- **Monitoring:** CloudWatch, Structured Logging
- **Development:** Docker Compose, Hot Reload

## 🔐 Security Features

- **Web3Auth Integration** - Social logins with Web3 wallet creation
- **JWT Authentication** - Stateless token-based auth
- **Role-Based Access** - Granular merchant/customer permissions  
- **Delegation Management** - Secure blockchain operation handling
- **API Key Authentication** - Service-to-service security
- **Encryption** - At rest and in transit data protection

## 🌐 Supported Networks

| Network | Mainnet | Testnet | Status |
|---------|---------|---------|--------|
| Ethereum | ✅ | Sepolia | Production |
| Polygon | ✅ | Mumbai | Production |  
| Arbitrum | ✅ | Sepolia | Production |
| Base | 🚧 | Sepolia | Coming Soon |

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](docs/contributing.md) for details.

### Development Workflow
1. **Fork & Clone:** Get the code locally
2. **Setup:** Run `npm run setup` for full installation
3. **Develop:** Make changes with hot reload
4. **Test:** Run `npm run test:all` before committing
5. **Submit:** Create a pull request

### Code Standards
- **Go:** Follow standard Go conventions, use `gofmt`
- **TypeScript:** ESLint + Prettier configuration
- **Commits:** Conventional commit messages
- **Documentation:** Update docs for new features

## 📊 Project Status

- **Version:** 2.0.0 (Monorepo)
- **Status:** Active Development
- **License:** MIT
- **Node.js:** ≥18.0.0 required
- **Go:** ≥1.21 required

### Recent Updates
- ✅ Migrated to Nx monorepo structure
- ✅ Implemented Web3Auth integration  
- ✅ Added MetaMask delegation support
- ✅ Enhanced Circle API integration
- 🚧 Mobile app development (planned)

## 🆘 Support

- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues
- **[GitHub Issues](https://github.com/your-org/cyphera-api/issues)** - Bug reports
- **[Discussions](https://github.com/your-org/cyphera-api/discussions)** - Questions
- **Documentation** - Comprehensive guides in `/docs`

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ by the Cyphera Team**

*For the latest updates and detailed documentation, visit our [documentation site](docs/) or check the individual service README files.*