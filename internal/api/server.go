// Package api provides HTTP server and handlers for the exchange API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/williansvarela/mb-clob/internal/domain"
	"github.com/williansvarela/mb-clob/internal/exchange"
)

// Server provides HTTP endpoints for interacting with the exchange service.
type Server struct {
	exchangeService *exchange.Service
	mux             *http.ServeMux
}

// CreateAccountRequest represents a request to create a new account.
type CreateAccountRequest struct {
	AccountID string `json:"account_id"`
}

// DepositRequest represents a request to deposit a specified amount of an asset into an account.
type DepositRequest struct {
	AccountID string `json:"account_id"`
	Asset     string `json:"asset"`
	Amount    int64  `json:"amount"`
}

// WithdrawRequest represents a request to withdraw a specified amount of an asset from an account.
type WithdrawRequest struct {
	AccountID string `json:"account_id"`
	Asset     string `json:"asset"`
	Amount    int64  `json:"amount"`
}

// PlaceOrderRequest represents a request to place a new order in the exchange.
type PlaceOrderRequest struct {
	AccountID string     `json:"account_id"`
	Side      domain.Side `json:"side"`
	Price     int64      `json:"price"`
	Quantity  int64      `json:"quantity"`
}

// CancelOrderRequest represents a request to cancel an existing order.
type CancelOrderRequest struct {
	OrderID string `json:"order_id"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a successful response from the API.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    any 		`json:"data,omitempty"`
}


// NewServer creates and returns a new Server instance with the provided exchange service.
func NewServer(exchangeService *exchange.Service) *Server {
	server := &Server{
		exchangeService: exchangeService,
		mux:             http.NewServeMux(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures the HTTP routes and their corresponding handler functions.
func (s *Server) setupRoutes() {
	// Health endpoint
	s.mux.HandleFunc("/health", s.handleHealth)

	// Account endpoints
	s.mux.HandleFunc("/accounts", s.handleAccounts)
	s.mux.HandleFunc("/accounts/", s.handleAccountDetails)
	s.mux.HandleFunc("/accounts/deposit", s.handleDeposit)
	s.mux.HandleFunc("/accounts/withdraw", s.handleWithdraw)
	s.mux.HandleFunc("/balances/", s.handleBalances)

	// Order endpoints
	s.mux.HandleFunc("/orders", s.handleOrders)
	s.mux.HandleFunc("/orders/", s.handleOrderDetails)

	// Market data endpoints
	s.mux.HandleFunc("/orderbook", s.handleOrderBook)
}


// Start launches the HTTP API server on the specified port.
func (s *Server) Start(port string) error {
	return http.ListenAndServe(":"+port, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	s.writeSuccess(w, health)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createAccount(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleAccountDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID := r.URL.Path[len("/accounts/"):]
	if accountID == "" {
		s.writeError(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	account, err := s.exchangeService.GetAccount(accountID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.writeSuccess(w, account)
}

func (s *Server) handleBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Format: /balances/{accountID}/{asset}
	path := r.URL.Path[len("/balances/"):]
	parts := splitPath(path)

	if len(parts) != 2 {
		s.writeError(w, http.StatusBadRequest, "Path should be /balances/{accountID}/{asset}")
		return
	}

	accountID := parts[0]
	asset := parts[1]

	balance, err := s.exchangeService.GetBalance(accountID, asset)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.writeSuccess(w, balance)
}

func (s *Server) handleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.exchangeService.Deposit(req.AccountID, req.Asset, req.Amount); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeSuccess(w, map[string]string{"message": "Deposit successful"})
}

func (s *Server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.exchangeService.Withdraw(req.AccountID, req.Asset, req.Amount); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeSuccess(w, map[string]string{"message": "Withdrawal successful"})
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.placeOrder(w, r)
	case http.MethodDelete:
		s.cancelOrder(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleOrderDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	orderID := r.URL.Path[len("/orders/"):]
	if orderID == "" {
		s.writeError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	order, err := s.exchangeService.GetOrder(orderID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.writeSuccess(w, order)
}

func (s *Server) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	snapshot := s.exchangeService.GetOrderBook()
	s.writeSuccess(w, snapshot)
}


func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AccountID == "" {
		s.writeError(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	if err := s.exchangeService.CreateAccount(req.AccountID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeSuccess(w, map[string]string{"message": "Account created successfully"})
}

func (s *Server) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	order, err := s.exchangeService.PlaceOrder(req.AccountID, req.Side, req.Price, req.Quantity)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeSuccess(w, order)
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.exchangeService.CancelOrder(req.OrderID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeSuccess(w, map[string]string{"message": "Order cancelled successfully"})
}

func (s *Server) writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
	})
}

func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var parts []string
	start := 0

	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}

	if start < len(path) {
		parts = append(parts, path[start:])
	}

	return parts
}
