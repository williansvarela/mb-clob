# Central Limit Order Book

A simplified Central Limit Order Book implementation for cryptocurrency trading. This system provides a simplified trading engine with order matching.

## 🎯 Project Overview

This is a simplified matching engine designed to handle cryptocurrency trading operations with focus on BTC/BRL trading pairs. It implements a Central Limit Order Book with price-time priority matching, trade execution, and account management.

## 🏗️ Architecture

The system follows a architecture pattern with separated concerns:

```
mb-clob/
├── cmd/                   # Application entry point
│   └── main.go            # Main server and demo setup
├── internal/              # Application core
│   ├── api/               # HTTP server and handlers
│   ├── account/           # Account management service
│   ├── domain/            # Core types and models
│   ├── exchange/          # Exchange orchestration service
│   ├── matching/          # Order matching engine
│   └── orderbook/         # Order book data structure
└── pkg/                   # Public packages
```

### Core Components

1. **Exchange Service** (`internal/exchange/`): Orchestrates all trading operations
2. **Matching Engine** (`internal/matching/`): Processes orders and executes trades
3. **Order Book** (`internal/orderbook/`): Maintains buy/sell order queues
4. **Account Service** (`internal/account/`): Manages user accounts and balances
5. **API Server** (`internal/api/`): Provides HTTP endpoints for client interaction

## 🚀 Getting Started

### Prerequisites

- Go 1.26.1 or later
- Make (optional, for using Makefile)

### Building the Project

Using Make:
```bash
make build
```

Or directly with Go:
```bash
go build -o bin/mb-clob ./cmd/main.go
```

### Running the Application

Using Make:
```bash
make run
```

Or directly:
```bash
./bin/mb-clob
# or
go run cmd/main.go
```

The server will start on `http://localhost:8080` with demo accounts and initial orders pre-loaded.

## Docker

### Docker Build

Build the Docker image using make

```bash
make docker-build
```

or directly:

```bash
docker build -t IMAGE_NAME .
```

### Running with Docker

```bash
# Run container in foreground
make docker-run

# Run container in background
make docker-run-bg
```

### Stop and Clean
```bash
# Stop container
make docker-stop

# Clean all Docker resources
make docker-clean
```

### Initial Demo Data

The application automatically sets up demo accounts with initial balances:
- **Accounts**: `alice`, `pedro`, `bruno`, `diana`
- **Initial BTC Balance**: 1.0 BTC per account
- **Initial BRL Balance**: R$ 150,000 per account
- **Sample Orders**: Pre-placed orders to demonstrate the order book

## 📚 API Documentation

The API provides comprehensive endpoints for trading operations:

### Health Check
```bash
GET /health
curl http://localhost:8080/health
```

### Account Management
```bash
# Create account
POST /accounts
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_id": "newuser"}'

# Get account details
GET /accounts/{accountID}
curl http://localhost:8080/accounts/alice

# Get balance
GET /balances/{accountID}/{asset}
curl http://localhost:8080/balances/alice/BTC
curl http://localhost:8080/balances/alice/BRL

# Deposit funds
POST /accounts/deposit
curl -X POST http://localhost:8080/accounts/deposit \
  -H "Content-Type: application/json" \
  -d '{"account_id": "alice", "asset": "BTC", "amount": 50000000}'

# Withdraw funds
POST /accounts/withdraw
curl -X POST http://localhost:8080/accounts/withdraw \
  -H "Content-Type: application/json" \
  -d '{"account_id": "alice", "asset": "BRL", "amount": 1000000}'
```

### Trading Operations
```bash
# Place order
POST /orders
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"account_id": "alice", "side": 1, "price": 30000000, "quantity": 10000000}'

# Cancel order
DELETE /orders
curl -X DELETE http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"order_id": "order_123"}'

# Get order details
GET /orders/{orderID}
curl http://localhost:8080/orders/order_123
```

### Market Data
```bash
# Get order book
GET /orderbook
curl http://localhost:8080/orderbook
```

## 🧪 Testing and Verification

### Manual Testing

1. **Start the application**:
   ```bash
   make run
   ```

2. **Check system health**:
   ```bash
   curl http://localhost:8080/health
   ```

3. **View initial order book**:
   ```bash
   curl http://localhost:8080/orderbook
   ```

4. **Check account balances**:
   ```bash
   curl http://localhost:8080/balances/alice/BTC
   curl http://localhost:8080/balances/pedro/BRL
   ```

5. **Place a test order** (Alice sells 0.1 BTC at R$ 300,000):
   ```bash
   curl -X POST http://localhost:8080/orders \
     -H "Content-Type: application/json" \
     -d '{"account_id": "alice", "side": 1, "price": 30000000, "quantity": 10000000}'
   ```

6. **Place a matching order** (Bob buys 0.1 BTC at R$ 300,000):
   ```bash
   curl -X POST http://localhost:8080/orders \
     -H "Content-Type: application/json" \
     -d '{"account_id": "pedro", "side": 0, "price": 30000000, "quantity": 10000000}'
   ```

7. **Verify trade execution**:
   ```bash
   curl http://localhost:8080/balances/alice/BTC
   curl http://localhost:8080/balances/pedro/BTC
   ```

### Expected Behavior

- Orders should match when buy price ≥ sell price
- Balances should update automatically after trades
- Order book should reflect current pending orders

## 📝 Assumptions

- **Instrument**: The system assume instrument is always 'BTC/BRL'.
- **Single Asset Pair**: This implementation focuses on one pair (e.g., BTC/BRL).

## 🔧 Implementation Details

### Precision and Units

The system uses integer arithmetic to avoid floating-point precision issues:

- **BTC Amounts**: Expressed in satoshis (1 BTC = 100,000,000 satoshis)
- **BRL Amounts**: Expressed in centavos (R$ 1.00 = 100 centavos)
- **Prices**: BRL centavos per satoshi

**Example Conversions**:
- 0.5 BTC = 50,000,000 satoshis
- R$ 300,000 = 30,000,000 centavos
- Price of R$ 300,000/BTC = 30,000,000 centavos per 100,000,000 satoshis

### Order Matching Algorithm

The matching engine implements **Price-Time Priority**:

1. **Algorithmic Complexity:**: The order book utilizes **Heaps** (Max-Heap for Bids, Min-Heap for Asks). This provides **O(1)** access to the "Best" price (top of the book) and **O(log N)** complexity for order insertions and cancellations.
2. **Price Priority**: Better prices are matched first
   - Buy orders: Higher prices have priority
   - Sell orders: Lower prices have priority

3. **Time Priority**: Among orders at the same price, earlier orders are matched first

4. **Partial Fills**: Large orders can be filled by multiple smaller orders

### Concurrency Model

- **Asynchronous Processing**: Orders are processed in a separate goroutine
- **Thread Safety**: All shared data structures use proper synchronization
- **Order Queue**: Incoming orders are queued for sequential processing
- **Real-time Updates**: Balance and order book updates happen immediately

### Error Handling

The system includes comprehensive error handling for:
- Insufficient balance validation
- Invalid order parameters
- Account existence verification
- Concurrent access protection
- Graceful shutdown handling

## 🎨 Design Decisions

### Why Integer Arithmetic?
- **Precision**: Avoids floating-point rounding errors
- **Consistency**: Ensures exact calculations for financial operations
- **Performance**: Integer operations are faster than floating-point

### Why Separate Services?
- **Modularity**: Each component has a single responsibility
- **Testability**: Services can be tested independently
- **Maintainability**: Clear boundaries between different concerns
- **Scalability**: Components can be scaled independently

### Why In-Memory Storage?
- **Performance**: Maximum speed for high-frequency trading
- **Simplicity**: No database configuration or management
- **Demonstration**: Suitable for prototyping and demonstration
- **Future Extension**: Can be extended with persistent storage

## 🔄 Order States and Lifecycle

Orders progress through the following states:

1. **Pending**: Order received, waiting for matching
2. **Partially Filled**: Order partially matched with counter-orders
3. **Filled**: Order completely matched
4. **Cancelled**: Order manually cancelled before completion

## 🚦 Side Enumeration

- **Buy Orders**: `side: 0` 
- **Sell Orders**: `side: 1`

## 🛠️ Development Commands

```bash
# Build the project
make build

# Run the application
make run

# Clean build artifacts
make clean

# Run with Go directly
go run cmd/main.go
```

## 🔮 Future Enhancements

- **Persistent Storage**: Add database integration for data persistence
- **Self-Trade Prevention**: Logic to reject orders where the Maker and Taker are the same AccountID.
- **Market Orders**: Adding support for orders that execute immediately at the best available price.
- **WebSocket API**: Real-time order book and trade streams
- **Multiple Trading Pairs**: Support for various cryptocurrency pairs
- **Rate Limiting**: API rate limiting and abuse prevention
- **Observability and Monitoring**: Add observability tool integration
- **Unit Tests**: Comprehensive test coverage
- **Authentication**: User authentication and authorization

## 📄 License

This project is for educational and demonstration purposes.