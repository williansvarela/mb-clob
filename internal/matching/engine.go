// Package matching implements the core matching engine for processing and executing orders in the order book.
package matching

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/williansvarela/mb-clob/internal/account"
	"github.com/williansvarela/mb-clob/internal/domain"
	"github.com/williansvarela/mb-clob/internal/orderbook"
)

// Engine represents the matching engine responsible for processing orders and executing trades
type Engine struct {
	mu             sync.RWMutex
	orderBook      *orderbook.OrderBook
	accountService *account.Service
	tradeHistory   []domain.Trade
	orderQueue     chan *domain.Order
	stopCh         chan struct{}
	tradeCallback  func(domain.Trade) // Callback para notificar trades
}

// NewEngine creates a new instance of the matching engine
func NewEngine(orderBook *orderbook.OrderBook, accountService *account.Service) *Engine {
	return &Engine{
		orderBook:      orderBook,
		accountService: accountService,
		tradeHistory:   make([]domain.Trade, 0),
		orderQueue:     make(chan *domain.Order, 1000),
		stopCh:         make(chan struct{}),
	}
}

// SetTradeCallback sets a callback function to be called when trades occur
func (e *Engine) SetTradeCallback(callback func(domain.Trade)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tradeCallback = callback
}

// Start starts the matching engine as a goroutine
func (e *Engine) Start() {
	go e.run()
}

// Stop stops the matching engine
func (e *Engine) Stop() {
	close(e.stopCh)
}

// SubmitOrder submits an order for processing
func (e *Engine) SubmitOrder(order *domain.Order) error {
	if err := e.validateOrder(order); err != nil {
		return err
	}

	if err := e.lockRequiredFunds(order); err != nil {
		return err
	}

	select {
	case e.orderQueue <- order:
		return nil
	default:
		e.unlockOrderFunds(order)
		return fmt.Errorf("order queue is full")
	}
}

// CancelOrder cancels an order
func (e *Engine) CancelOrder(orderID string) error {
	order, err := e.orderBook.CancelOrder(orderID)
	if err != nil {
		return err
	}

	return e.unlockOrderFunds(order)
}

// GetTrades returns the trade history
func (e *Engine) GetTrades() []domain.Trade {
	e.mu.RLock()
	defer e.mu.RUnlock()

	trades := make([]domain.Trade, len(e.tradeHistory))
	copy(trades, e.tradeHistory)
	return trades
}

// run is the main loop of the matching engine
func (e *Engine) run() {
	for {
		select {
		case order := <-e.orderQueue:
			e.processOrder(order)
		case <-e.stopCh:
			return
		}
	}
}

// processOrder processes an individual order
func (e *Engine) processOrder(order *domain.Order) {
	e.orderBook.AddOrder(order)

	for {
		if !e.attemptMatch(order) {
			break
		}

		// Stop processing if the order is fully filled
		if order.Remaining == 0 {
			order.Status = domain.Filled
			break
		}
	}
}

// attemptMatch attempts to match the incoming order with existing orders
func (e *Engine) attemptMatch(incomingOrder *domain.Order) bool {
	var matchingOrder *domain.Order

	if incomingOrder.Side == domain.Buy {
		matchingOrder = e.orderBook.PeekBestSell()
		if matchingOrder == nil || matchingOrder.Price > incomingOrder.Price {
			return false // No match possible
		}
	} else {
		matchingOrder = e.orderBook.PeekBestBuy()
		if matchingOrder == nil || matchingOrder.Price < incomingOrder.Price {
			return false // No match possible
		}
	}

	tradeQuantity := min(incomingOrder.Remaining, matchingOrder.Remaining)
	tradePrice := matchingOrder.Price // Price of the order that was in the book

	trade := domain.Trade{
		ID:        generateTradeID(),
		Price:     tradePrice,
		Quantity:  tradeQuantity,
		Timestamp: time.Now(),
	}

	if incomingOrder.Side == domain.Buy {
		trade.BuyOrderID = incomingOrder.ID
		trade.SellOrderID = matchingOrder.ID
		trade.BuyerID = incomingOrder.AccountID
		trade.SellerID = matchingOrder.AccountID
	} else {
		trade.BuyOrderID = matchingOrder.ID
		trade.SellOrderID = incomingOrder.ID
		trade.BuyerID = matchingOrder.AccountID
		trade.SellerID = incomingOrder.AccountID
	}

	if err := e.executeTrade(trade); err != nil {
		slog.Error("Trade execution failed", "error", err)
		return false
	}

	incomingOrder.Remaining -= tradeQuantity
	matchingOrder.Remaining -= tradeQuantity

	if matchingOrder.Remaining == 0 {
		matchingOrder.Status = domain.Filled
		if incomingOrder.Side == domain.Buy {
			e.orderBook.PopBestSell()
		} else {
			e.orderBook.PopBestBuy()
		}
	} else {
		matchingOrder.Status = domain.PartiallyFilled
	}

	if incomingOrder.Remaining == 0 {
		incomingOrder.Status = domain.Filled
	} else {
		incomingOrder.Status = domain.PartiallyFilled
	}

	e.mu.Lock()
	e.tradeHistory = append(e.tradeHistory, trade)
	e.mu.Unlock()

	if e.tradeCallback != nil {
		e.tradeCallback(trade)
	}

	return true
}

// executeTrade executes the asset transfers for a trade
func (e *Engine) executeTrade(trade domain.Trade) error {
	// Determine the assets (assuming BTC/BRL for now)
	baseAsset := "BTC"
	quoteAsset := "BRL"

	totalQuoteAmount := (trade.Price * trade.Quantity) / domain.Satoshi

	if err := e.accountService.TransferLockedFunds(trade.SellerID, trade.BuyerID, baseAsset, trade.Quantity); err != nil {
		return fmt.Errorf("failed to transfer %s: %v", baseAsset, err)
	}

	if err := e.accountService.TransferLockedFunds(trade.BuyerID, trade.SellerID, quoteAsset, totalQuoteAmount); err != nil {
		e.accountService.TransferLockedFunds(trade.BuyerID, trade.SellerID, baseAsset, trade.Quantity)
		return fmt.Errorf("failed to transfer %s: %v", quoteAsset, err)
	}

	return nil
}

// validateOrder validates an order before processing it
func (e *Engine) validateOrder(order *domain.Order) error {
	if order.ID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}
	if order.AccountID == "" {
		return fmt.Errorf("account ID cannot be empty")
	}
	if order.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if order.Side != domain.Buy && order.Side != domain.Sell {
		return fmt.Errorf("invalid order side")
	}
	if order.Type != domain.Limit {
		return fmt.Errorf("Unsupported order type: only limit orders are supported for now")
	}
	if order.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if order.Symbol != "BTC/BRL" {
		return fmt.Errorf("unsupported symbol: only BTC/BRL is supported for now")
	}
	return nil
}

// lockRequiredFunds locks the required funds for an order
func (e *Engine) lockRequiredFunds(order *domain.Order) error {
	if order.Side == domain.Buy {
		totalAmount := (order.Price * order.Quantity) / domain.Satoshi
		return e.accountService.LockFunds(order.AccountID, "BRL", totalAmount)
	} else {
		return e.accountService.LockFunds(order.AccountID, "BTC", order.Quantity)
	}
}

// unlockOrderFunds unlocks the locked funds for an order
func (e *Engine) unlockOrderFunds(order *domain.Order) error {
	if order.Side == domain.Buy {
		remainingValue := (order.Price * order.Remaining) / domain.Satoshi
		return e.accountService.UnlockFunds(order.AccountID, "BRL", remainingValue)
	} else {
		return e.accountService.UnlockFunds(order.AccountID, "BTC", order.Remaining)
	}
}

// min returns the smaller of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// generateTradeID generates a unique ID for a trade
func generateTradeID() string {
	return fmt.Sprintf("trade_%d", time.Now().UnixNano())
}
