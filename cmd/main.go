// Package main implements the entry point for the Crypto Exchange Central Limit Order Book application.
package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/williansvarela/mb-clob/internal/api"
	"github.com/williansvarela/mb-clob/internal/domain"
	"github.com/williansvarela/mb-clob/internal/exchange"
)

func main() {
	slog.Info("Starting Crypto Exchange Central Limit Order Book")

	slog.Debug("Initializing exchange service...")
	exchangeService := exchange.NewService("BTC/BRL")
	exchangeService.Start()

	if err := setupInitialData(exchangeService); err != nil {
		log.Fatalf("Failed to setup initial data: %v", err)
	}

	apiServer := api.NewServer(exchangeService)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := "8080"
		fmt.Printf("\nExchange API Server starting on http://localhost:%s\n", port)
		fmt.Println("\nAvailable endpoints:")
		printAPIEndpoints(port)

		if err := apiServer.Start(port); err != nil {
			log.Fatalf("API server failed to start: %v", err)
		}
	}()

	<-signalCh

	slog.Info("Shutdown signal received, stopping services...")
	exchangeService.Stop()
	slog.Info("All services stopped successfully")
}

// setupInitialData populates the exchange with initial accounts, balances, and orders for demonstration purposes.
func setupInitialData(exchangeService *exchange.Service) error {
	fmt.Println("\nSetting up initial demo data...")

	// Create example accounts
	accounts := []string{"alice", "bob", "charlie", "diana"}

	for _, accountID := range accounts {
		if err := exchangeService.CreateAccount(accountID); err != nil {
			return fmt.Errorf("failed to create account %s: %v", accountID, err)
		}

		// Deposit initial funds
		// BTC (represented in satoshis: 1 BTC = 100,000,000 satoshis)
		if err := exchangeService.Deposit(accountID, "BTC", 100000000); err != nil { // 1 BTC
			return fmt.Errorf("failed to deposit BTC to %s: %v", accountID, err)
		}

		// BRL (represented in cents: R$ 100,000.00 = 10,000,000 cents)
		if err := exchangeService.Deposit(accountID, "BRL", 15000000); err != nil { // R$ 150,000
			return fmt.Errorf("failed to deposit BRL to %s: %v", accountID, err)
		}

		fmt.Printf("   ✓ Account '%s' created with 1 BTC and R$ 150,000\n", accountID)
	}

	// Place some initial orders to demonstrate the order book
	fmt.Println("\nPlacing initial orders...")

	// Alice: Sell orders
	if _, err := exchangeService.PlaceOrder("alice", domain.Sell, 30000000, 25000000); err != nil { // Sell 0.25 BTC at R$ 300,000
		return fmt.Errorf("failed to place Alice's sell order: %v", err)
	}
	if _, err := exchangeService.PlaceOrder("alice", domain.Sell, 31000000, 15000000); err != nil { // Sell 0.15 BTC at R$ 310,000
		return fmt.Errorf("failed to place Alice's sell order: %v", err)
	}

	// Bob: Buy orders
	if _, err := exchangeService.PlaceOrder("bob", domain.Buy, 29000000, 15000000); err != nil { // Buy 0.15 BTC at R$ 290,000
		return fmt.Errorf("failed to place Bob's buy order: %v", err)
	}
	if _, err := exchangeService.PlaceOrder("bob", domain.Buy, 28500000, 20000000); err != nil { // Buy 0.2 BTC at R$ 285,000
		return fmt.Errorf("failed to place Bob's buy order: %v", err)
	}

	// Charlie: Mix of orders
	if _, err := exchangeService.PlaceOrder("charlie", domain.Buy, 29500000, 10000000); err != nil { // Buy 0.1 BTC at R$ 295,000
		return fmt.Errorf("failed to place Charlie's buy order: %v", err)
	}

	fmt.Println("Initial orders placed successfully!")

	showOrderBookStatus(exchangeService)

	return nil
}

// showOrderBookStatus shows the current state of the order book
func showOrderBookStatus(exchangeService *exchange.Service) {
	fmt.Println("\nCurrent Order Book Status:")

	snapshot := exchangeService.GetOrderBook()

	fmt.Println("\nSELL ORDERS (Ask):")
	for i, order := range snapshot.SellOrders {
		if i >= 5 {
			break
		}
		fmt.Printf("   %s: %.8f BTC @ R$ %.2f (Account: %s)\n",
			order.ID[:8]+"...",
			float64(order.Remaining)/100000000,
			float64(order.Price)/100,
			order.AccountID)
	}

	bestBuy, bestSell := exchangeService.GetBestPrices()
	if bestSell > 0 && bestBuy > 0 {
		spread := bestSell - bestBuy
		fmt.Printf("\nSpread: R$ %.2f (%.2f%%)\n",
			float64(spread)/100,
			float64(spread)/float64(bestSell)*100)
	}

	fmt.Println("\nBUY ORDERS (Bid):")
	for i, order := range snapshot.BuyOrders {
		if i >= 5 {
			break
		}
		fmt.Printf("   %s: %.8f BTC @ R$ %.2f (Account: %s)\n",
			order.ID[:8]+"...",
			float64(order.Remaining)/100000000,
			float64(order.Price)/100,
			order.AccountID)
	}

	trades := exchangeService.GetTrades()
	if len(trades) > 0 {
		fmt.Printf("\nTotal Trades Executed: %d\n", len(trades))

		lastTrade := trades[len(trades)-1]
		fmt.Printf("   Last: %.8f BTC @ R$ %.2f between %s <-> %s\n",
			float64(lastTrade.Quantity)/100000000,
			float64(lastTrade.Price)/100,
			lastTrade.BuyerID,
			lastTrade.SellerID)
	}
}

// printAPIEndpoints shows the available endpoints
func printAPIEndpoints(port string) {
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	fmt.Printf("   📋 Health Check:        GET  %s/health\n", baseURL)
	fmt.Printf("   👤 Create Account:      POST %s/accounts\n", baseURL)
	fmt.Printf("   💰 Account Details:     GET  %s/accounts/{accountID}\n", baseURL)
	fmt.Printf("   💳 Deposit:             POST %s/accounts/deposit\n", baseURL)
	fmt.Printf("   🏧 Withdraw:            POST %s/accounts/withdraw\n", baseURL)
	fmt.Printf("   💵 Balance:             GET  %s/balances/{accountID}/{asset}\n", baseURL)
	fmt.Printf("   📝 Place Order:         POST %s/orders\n", baseURL)
	fmt.Printf("   ❌ Cancel Order:        DELETE %s/orders\n", baseURL)
	fmt.Printf("   📊 Order Details:       GET  %s/orders/{orderID}\n", baseURL)
	fmt.Printf("   📚 Order Book:          GET  %s/orderbook\n", baseURL)

	fmt.Println("\nExample requests:")
	fmt.Printf("   curl %s/health\n", baseURL)
	fmt.Printf("   curl %s/orderbook\n", baseURL)
	fmt.Printf("   curl %s/balances/alice/BTC\n", baseURL)
	curl := `curl -X POST %s/orders -H "Content-Type: application/json" -d '{"account_id":"alice","side":1,"price":30000000,"quantity":10000000}'`
	fmt.Printf("   "+curl+"\n", baseURL)

	fmt.Println("\nNote: Prices and quantities are in the smallest unit:")
	fmt.Println("   - BTC: 1 BTC = 100,000,000 satoshis")
	fmt.Println("   - BRL: R$ 1.00 = 100 centavos")
	fmt.Println("   - Side: 0 = BUY, 1 = SELL")
}
