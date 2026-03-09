# Decimal Package Use Case

This document provides a practical example of how to use the `decimal` and `null.ToNullDecimal` packages in a financial service. This example is based on the business rules outlined in the CRM specifications (e.g., handling a deposit, strict 6-digit precision, non-negative wallet balances).

## Example: Wallet Service (TA-001 DEPOSIT_PSP)

```go
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/decimal"
	"github.com/samsonnaze5/aeternixth-go-lib/null"
)

// Wallet represents a database record for a given user wallet.
type Wallet struct {
	ID      string
	Balance decimal.Decimal     // Strict 6-digit precision required by CRM
	Bonus   decimal.NullDecimal // Nullable column in the DB
}

// WalletService handles financial operations and interactions with the database.
type WalletService struct {
	db *sql.DB
}

// DepositPSP (TA-001) adds funds to a Default Client Wallet.
func (s *WalletService) DepositPSP(ctx context.Context, walletID string, amountStr string, bonusStr *string) error {
	// 1. Parse the incoming amount with exact precision (avoids float64 inaccuracies)
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return fmt.Errorf("invalid deposit amount: %w", err)
	}

	// 2. Business Logic: Amount must be strictly greater than zero
	if amount.LessThanOrEqual(decimal.Zero()) {
		return fmt.Errorf("deposit amount must be positive")
	}

	// 3. Fetch the current wallet balance 
	// (In a production app, this should be executed inside a DB transaction with a row lock, e.g., "FOR UPDATE")
	wallet, err := s.getWallet(ctx, walletID)
	if err != nil {
		return err
	}

	// 4. Calculate the new balance: use exact math with decimal.Add() instead of standard operators
	newBalance := wallet.Balance.Add(amount)

	// 5. Constraints: Wallets cannot drop below $0.000000 
	if newBalance.LessThan(decimal.Zero()) {
		return fmt.Errorf("wallet balance cannot be negative")
	}

	// 6. Handle optional bonus string using our pointer converter
	var parsedBonus *decimal.Decimal
	if bonusStr != nil {
		bonusAmt, err := decimal.NewFromString(*bonusStr)
		if err == nil {
			parsedBonus = &bonusAmt
		}
	}
	
	// Convert *decimal.Decimal pointer to a database-friendly decimal.NullDecimal
	nullBonus := null.ToNullDecimal(parsedBonus)

	// 7. Save to the Database
	query := `
		UPDATE wallets 
		SET balance = $1, bonus_balance = $2 
		WHERE id = $3
	`
	_, err = s.db.ExecContext(ctx, query, newBalance, nullBonus, walletID)
	return err
}

// getWallet simulates fetching a wallet from the database.
func (s *WalletService) getWallet(ctx context.Context, id string) (*Wallet, error) {
	// Mock returning a wallet with exactly $100.50 Initial Balance
	return &Wallet{
		ID:      id,
		Balance: decimal.RequireFromString("100.500000"), // Maintaining 6-digit precision format
		Bonus:   null.ToNullDecimal(nil),                 // Starts as NULL in the database
	}, nil
}
```

## Key Takeaways

1. **Never use `float64` for math.** Always utilize functions like `.Add()`, `.Sub()`, `.Mul()`, `.Div()`, etc.
2. **Comparing Values.** Avoid standard `>` or `<` operators. Use `.LessThan()`, `.GreaterThan()`, `.Equal()`, and compare values against `decimal.Zero()`.
3. **Database Nulls.** When retrieving optional values from a JSON request (where a field might be a pointer like `*string`), parse it to a `*decimal.Decimal` and use `null.ToNullDecimal(&val)` to securely save it to your database without risk of nil panic or precision loss.
