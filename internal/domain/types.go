// Package domain contains core types and domain models for the CLOB system.
package domain

import (
	"time"
)

// Satoshi is the smallest unit of Bitcoin (1 BTC = 100 million satoshis)
const Satoshi int64 = 100000000

// Side represents the side of an order (buy or sell)
type Side int

// Buy represents a buy order side.
// Sell represents a sell order side.
const (
	Buy Side = iota
	Sell
)

var sideToString = map[Side]string{
	Buy:  "BUY",
	Sell: "SELL",
}

func (s Side) String() string {
	if str, ok := sideToString[s]; ok {
		return str
	}
	return "UNKNOWN"
}

// OrderType represents the type of an order (limit or market)
type OrderType int

// Limit represents a limit order type.
// Market represents a market order type.
const (
	Limit OrderType = iota
	Market
)

// OrderStatus represents the status of an order
type OrderStatus int

// Pending indicates that the order has been created but not yet filled or cancelled.
// PartiallyFilled indicates that the order has been partially filled.
// Filled indicates that the order has been completely filled.
// Cancelled indicates that the order has been cancelled.
const (
	Pending OrderStatus = iota
	PartiallyFilled
	Filled
	Cancelled
)

func (os OrderStatus) String() string {
	switch os {
	case Pending:
		return "PENDING"
	case PartiallyFilled:
		return "PARTIALLY_FILLED"
	case Filled:
		return "FILLED"
	case Cancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

// Order represents an order in the system
type Order struct {
	ID        string      `json:"id"`
	AccountID string      `json:"account_id"`
	Side      Side        `json:"side"`
	Price     int64       `json:"price"`
	Quantity  int64       `json:"quantity"`
	Remaining int64       `json:"remaining"`
	Status    OrderStatus `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Type      OrderType   `json:"type"`
	Symbol    string      `json:"symbol"`
}

// Trade represents a matched trade between two orders
type Trade struct {
	ID          string    `json:"id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// Balance represents the balance of an account
type Balance struct {
	Asset  string `json:"asset"`
	Amount int64  `json:"amount"`
	Locked int64  `json:"locked"`
}

// Account represents a user account with asset balances
type Account struct {
	ID       string             `json:"id"`
	Balances map[string]Balance `json:"balances"`
}


// OrderBookSnapshot represents a snapshot of the order book at a given time
type OrderBookSnapshot struct {
	BuyOrders  []Order   `json:"buy_orders"`
	SellOrders []Order   `json:"sell_orders"`
	Timestamp  time.Time `json:"timestamp"`
}
