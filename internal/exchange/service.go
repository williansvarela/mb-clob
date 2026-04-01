// Package exchange provides the main service and orchestration logic for the exchange operations.
package exchange

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/williansvarela/mb-clob/internal/account"
	"github.com/williansvarela/mb-clob/internal/domain"
	"github.com/williansvarela/mb-clob/internal/matching"
	"github.com/williansvarela/mb-clob/internal/orderbook"
)

// Service is the main service that orchestrates the exchange operations
type Service struct {
	accountService *account.Service
	orderBook      *orderbook.OrderBook
	matchingEngine *matching.Engine
	instrument     string // Ex: "BTC/BRL"
}

// NewService creates a new instance of the exchange service
func NewService(instrument string) *Service {
	accountService := account.NewService()
	orderBook := orderbook.NewOrderBook()
	matchingEngine := matching.NewEngine(orderBook, accountService)

	service := &Service{
		accountService: accountService,
		orderBook:      orderBook,
		matchingEngine: matchingEngine,
		instrument:     instrument,
	}

	matchingEngine.SetTradeCallback(service.onTrade)

	return service
}

// Start starts the exchange service (matching engine)
func (s *Service) Start() {
	slog.Info("Starting exchange service", "instrument", s.instrument)
	s.matchingEngine.Start()
}

// Stop stops the exchange service (matching engine)
func (s *Service) Stop() {
	slog.Info("Stopping exchange service", "instrument", s.instrument)
	s.matchingEngine.Stop()
}

// CreateAccount creates a new account
func (s *Service) CreateAccount(accountID string) error {
	return s.accountService.CreateAccount(accountID)
}

// GetAccount returns account information
func (s *Service) GetAccount(accountID string) (*domain.Account, error) {
	return s.accountService.GetAccount(accountID)
}

// Deposit adds funds to an account
func (s *Service) Deposit(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}

	err := s.accountService.Credit(accountID, asset, amount)
	if err == nil {
		slog.Info("Deposited funds", "amount", amount, "asset", asset, "account", accountID)
	}
	return err
}

// Withdraw removes funds from an account
func (s *Service) Withdraw(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive")
	}

	err := s.accountService.Debit(accountID, asset, amount)
	if err == nil {
		slog.Info("Withdrawn funds", "amount", amount, "asset", asset, "account", accountID)
	}
	return err
}

// PlaceOrder places a new order in the system
func (s *Service) PlaceOrder(accountID string, side domain.Side, price, quantity int64) (*domain.Order, error) {
	order := &domain.Order{
		ID:        generateOrderID(),
		AccountID: accountID,
		Side:      side,
		Price:     price,
		Quantity:  quantity,
		Timestamp: time.Now(),
	}

	err := s.matchingEngine.SubmitOrder(order)
	if err != nil {
		return nil, err
	}

	slog.Info("Order placed", "order_id", order.ID, "side", side, "quantity", quantity, "price", price, "account", accountID)

	return order, nil
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(orderID string) error {
	err := s.matchingEngine.CancelOrder(orderID)
	if err == nil {
		slog.Info("Order cancelled", "order_id", orderID)
	}
	return err
}

// GetOrder returns information about a specific order
func (s *Service) GetOrder(orderID string) (*domain.Order, error) {
	return s.orderBook.GetOrder(orderID)
}

// GetOrderBook returns the current state of the order book
func (s *Service) GetOrderBook() domain.OrderBookSnapshot {
	return s.orderBook.GetSnapshot()
}

// GetBalance returns the balance of a specific asset
func (s *Service) GetBalance(accountID, asset string) (domain.Balance, error) {
	return s.accountService.GetBalance(accountID, asset)
}

// GetBestPrices returns the best buy and sell prices
func (s *Service) GetBestPrices() (bestBuy, bestSell int64) {
	return s.orderBook.GetBestPrices()
}

// GetTrades returns the trade history
func (s *Service) GetTrades() []domain.Trade {
	return s.matchingEngine.GetTrades()
}

// onTrade is called when a trade is executed
func (s *Service) onTrade(trade domain.Trade) {
	slog.Info("Trade executed", "trade_id", trade.ID, "quantity", trade.Quantity, "price", trade.Price, "buyer", trade.BuyerID, "seller", trade.SellerID)
}

// generateOrderID generates a unique ID for an order
func generateOrderID() string {
	return fmt.Sprintf("order_%d", time.Now().UnixNano())
}
