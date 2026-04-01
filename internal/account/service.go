// Package account provides services for managing accounts and balances.
package account

import (
	"fmt"
	"maps"
	"sync"

	"github.com/williansvarela/mb-clob/internal/domain"
)

// Service manages accounts and their balances
type Service struct {
	mu       sync.RWMutex
	accounts map[string]*domain.Account
}

// NewService creates a new instance of the account service
func NewService() *Service {
	return &Service{
		accounts: make(map[string]*domain.Account),
	}
}

// CreateAccount creates a new account
func (s *Service) CreateAccount(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[accountID]; exists {
		return fmt.Errorf("account %s already exists", accountID)
	}

	s.accounts[accountID] = &domain.Account{
		ID:       accountID,
		Balances: make(map[string]domain.Balance),
	}

	return nil
}

// copyAccount creates a deep copy of an account
func (s *Service) copyAccount(src *domain.Account) *domain.Account {
	account := &domain.Account{
		ID:       src.ID,
		Balances: make(map[string]domain.Balance),
	}

	maps.Copy(account.Balances, src.Balances)

	return account
}

// GetAccount returns an account's information
func (s *Service) GetAccount(accountID string) (*domain.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	accountCopy := s.copyAccount(account)

	return accountCopy, nil
}

// Credit adds funds to an account
func (s *Service) Credit(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	balance := account.Balances[asset]
	balance.Asset = asset
	balance.Amount += amount

	account.Balances[asset] = balance
	return nil
}

// Debit removes funds from an account
func (s *Service) Debit(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	balance, exists := account.Balances[asset]
	if !exists {
		return fmt.Errorf("insufficient funds: no %s balance", asset)
	}

	if balance.Amount < amount {
		return fmt.Errorf("insufficient funds: need %d but have %d", amount, balance.Amount)
	}

	balance.Amount -= amount
	account.Balances[asset] = balance
	return nil
}

// LockFunds locks funds for an order
func (s *Service) LockFunds(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	balance, exists := account.Balances[asset]
	if !exists {
		return fmt.Errorf("insufficient funds: no %s balance", asset)
	}

	available := balance.Amount - balance.Locked
	if available < amount {
		return fmt.Errorf("insufficient available funds: need %d but have %d available", amount, available)
	}

	balance.Locked += amount
	account.Balances[asset] = balance
	return nil
}

// UnlockFunds unlocks locked funds
func (s *Service) UnlockFunds(accountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	balance, exists := account.Balances[asset]
	if !exists {
		return fmt.Errorf("no %s balance found", asset)
	}

	if balance.Locked < amount {
		return fmt.Errorf("cannot unlock %d: only %d locked", amount, balance.Locked)
	}

	balance.Locked -= amount
	account.Balances[asset] = balance
	return nil
}

// TransferLockedFunds transfers locked funds (used in trades)
func (s *Service) TransferLockedFunds(fromAccountID, toAccountID, asset string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fromAccount, exists := s.accounts[fromAccountID]
	if !exists {
		return fmt.Errorf("from account %s not found", fromAccountID)
	}

	toAccount, exists := s.accounts[toAccountID]
	if !exists {
		return fmt.Errorf("to account %s not found", toAccountID)
	}

	fromBalance, exists := fromAccount.Balances[asset]
	if !exists {
		return fmt.Errorf("no %s balance in from account", asset)
	}

	if fromBalance.Locked < amount {
		return fmt.Errorf("insufficient locked funds: need %d but have %d locked", amount, fromBalance.Locked)
	}

	fromBalance.Amount -= amount
	fromBalance.Locked -= amount
	fromAccount.Balances[asset] = fromBalance

	toBalance := toAccount.Balances[asset]
	toBalance.Asset = asset
	toBalance.Amount += amount
	toAccount.Balances[asset] = toBalance

	return nil
}

// GetBalance returns the balance of a specific asset
func (s *Service) GetBalance(accountID, asset string) (domain.Balance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return domain.Balance{}, fmt.Errorf("account %s not found", accountID)
	}

	balance, exists := account.Balances[asset]
	if !exists {
		return domain.Balance{Asset: asset, Amount: 0, Locked: 0}, nil
	}

	return balance, nil
}
