// Package orderbook implements an in-memory order book for managing buy and sell orders.
package orderbook

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/williansvarela/mb-clob/internal/domain"
)

// BuyOrderHeap implements a max-heap for buy orders (highest price first, then FIFO by time)
type BuyOrderHeap []*domain.Order

// Len returns the number of orders in the heap
func (h BuyOrderHeap) Len() int { return len(h) }

// Less defines the ordering of orders in the heap
// Primarily by price (higher first), then by timestamp (older first) for FIFO
func (h BuyOrderHeap) Less(i, j int) bool {
	if h[i].Price == h[j].Price {
		return h[i].Timestamp.Before(h[j].Timestamp)
	}
	return h[i].Price > h[j].Price
}

// Swap exchanges the positions of two orders in the heap
func (h BuyOrderHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push adds a new order to the heap
func (h *BuyOrderHeap) Push(x any) {
	*h = append(*h, x.(*domain.Order))
}

// Pop removes and returns the top order from the heap
func (h *BuyOrderHeap) Pop() any {
	old := *h
	n := len(old)
	order := old[n-1]
	*h = old[0 : n-1]
	return order
}

// SellOrderHeap implements a min-heap for sell orders (lowest price first, then FIFO by time)
type SellOrderHeap []*domain.Order

// Len returns the number of orders in the heap
func (h SellOrderHeap) Len() int { return len(h) }

// Less defines the ordering of orders in the heap
// Primarily by price (lower first), then by timestamp (older first) for FIFO
func (h SellOrderHeap) Less(i, j int) bool {
	if h[i].Price == h[j].Price {
		return h[i].Timestamp.Before(h[j].Timestamp)
	}
	return h[i].Price < h[j].Price
}

// Swap exchanges the positions of two orders in the heap
func (h SellOrderHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push adds a new order to the heap
func (h *SellOrderHeap) Push(x any) {
	*h = append(*h, x.(*domain.Order))
}

// Pop removes and returns the top order from the heap
func (h *SellOrderHeap) Pop() any {
	old := *h
	n := len(old)
	order := old[n-1]
	*h = old[0 : n-1]
	return order
}

// OrderBook represents a order book for a single instrument (e.g., BTC/BRL)
type OrderBook struct {
	mu         sync.RWMutex
	buyOrders  *BuyOrderHeap
	sellOrders *SellOrderHeap
	orderIndex map[string]*domain.Order
}

// NewOrderBook creates a new order book
func NewOrderBook() *OrderBook {
	buyOrders := &BuyOrderHeap{}
	sellOrders := &SellOrderHeap{}
	heap.Init(buyOrders)
	heap.Init(sellOrders)

	return &OrderBook{
		buyOrders:  buyOrders,
		sellOrders: sellOrders,
		orderIndex: make(map[string]*domain.Order),
	}
}

// AddOrder adds an order to the book
func (ob *OrderBook) AddOrder(order *domain.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order.Status = domain.Pending
	order.Remaining = order.Quantity
	ob.orderIndex[order.ID] = order

	if order.Side == domain.Buy {
		heap.Push(ob.buyOrders, order)
	} else {
		heap.Push(ob.sellOrders, order)
	}
}

// CancelOrder removes an order from the book and returns it
func (ob *OrderBook) CancelOrder(orderID string) (*domain.Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.orderIndex[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	if order.Status != domain.Pending && order.Status != domain.PartiallyFilled {
		return nil, fmt.Errorf("cannot cancel order %s with status %s", orderID, order.Status)
	}

	order.Status = domain.Cancelled
	delete(ob.orderIndex, orderID)

	return order, nil
}

// GetOrder returns a specific order by ID
func (ob *OrderBook) GetOrder(orderID string) (*domain.Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	order, exists := ob.orderIndex[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	orderCopy := *order
	return &orderCopy, nil
}

// GetSnapshot returns a snapshot of the order book
func (ob *OrderBook) GetSnapshot() domain.OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	buyOrders := make([]domain.Order, 0, ob.buyOrders.Len())
	sellOrders := make([]domain.Order, 0, ob.sellOrders.Len())

	for _, order := range *ob.buyOrders {
		if order.Status == domain.Pending || order.Status == domain.PartiallyFilled {
			buyOrders = append(buyOrders, *order)
		}
	}

	for _, order := range *ob.sellOrders {
		if order.Status == domain.Pending || order.Status == domain.PartiallyFilled {
			sellOrders = append(sellOrders, *order)
		}
	}

	return domain.OrderBookSnapshot{
		BuyOrders:  buyOrders,
		SellOrders: sellOrders,
		Timestamp:  time.Now(),
	}
}

// PeekBestBuy returns the best buy order without removing it
func (ob *OrderBook) PeekBestBuy() *domain.Order {
	for ob.buyOrders.Len() > 0 {
		order := (*ob.buyOrders)[0]
		if order.Status == domain.Pending || order.Status == domain.PartiallyFilled {
			return order
		}
		// Remove cancelled/filled orders
		heap.Pop(ob.buyOrders)
		delete(ob.orderIndex, order.ID)
	}
	return nil
}

// PeekBestSell returns the best sell order without removing it
func (ob *OrderBook) PeekBestSell() *domain.Order {
	for ob.sellOrders.Len() > 0 {
		order := (*ob.sellOrders)[0]
		if order.Status == domain.Pending || order.Status == domain.PartiallyFilled {
			return order
		}
		// Remove cancelled/filled orders
		heap.Pop(ob.sellOrders)
		delete(ob.orderIndex, order.ID)
	}
	return nil
}

// PopBestBuy removes and returns the best buy order
func (ob *OrderBook) PopBestBuy() *domain.Order {
	order := ob.PeekBestBuy()
	if order != nil {
		heap.Pop(ob.buyOrders)
	}
	return order
}

// PopBestSell removes and returns the best sell order
func (ob *OrderBook) PopBestSell() *domain.Order {
	order := ob.PeekBestSell()
	if order != nil {
		heap.Pop(ob.sellOrders)
	}
	return order
}

// GetBestPrices returns the best buy and sell prices
func (ob *OrderBook) GetBestPrices() (bestBuy, bestSell int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bestBuyOrder := ob.PeekBestBuy()
	if bestBuyOrder != nil {
		bestBuy = bestBuyOrder.Price
	}

	bestSellOrder := ob.PeekBestSell()
	if bestSellOrder != nil {
		bestSell = bestSellOrder.Price
	}

	return bestBuy, bestSell
}
