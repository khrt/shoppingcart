package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Cart is a shopping cart of a user, holds theirs line items.
type Cart struct {
	ID        int64
	UserID    int64
	LineItems []*LineItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LineItem is an SKU item of a cart with a quantity multiplier.
type LineItem struct {
	ID        int64
	CartID    int64
	ProductID int64
	Quantity  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// storer describes Shopping Cart storage functions.
type storer interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (storer, error) // NOTE: an interesting point to discuss
	Commit() error

	CartCreate(ctx context.Context, cart *Cart) error
	CartWithItemsByCartID(ctx context.Context, cartID int64) (*Cart, error)
	CartEmpty(ctx context.Context, cartID int64) error

	LineItemsUpsert(ctx context.Context, cartID int64, items ...*LineItem) error
	LineItemRemove(ctx context.Context, cartID, itemID int64) error
}

// ShoppingCart holds business logic.
type ShoppingCart struct {
	storage storer
}

// CartCreate creates and persists a shopping cart, returns created cart with items if were any.
func (sc *ShoppingCart) CartCreate(ctx context.Context, userID int64, items []*LineItem) (*Cart, error) {
	cart := &Cart{
		UserID:    userID,
		LineItems: items,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		tx  storer
		err error
	)
	tx, err = sc.storage.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("tx: %w", err)
	}

	if err := tx.CartCreate(ctx, cart); err != nil {
		return nil, fmt.Errorf("cart: %w", err)
	}

	if err := tx.LineItemsUpsert(ctx, cart.ID, items...); err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return cart, nil
}

// CartShow returns the details of a cart.
func (sc *ShoppingCart) CartShow(ctx context.Context, cartID int64) (*Cart, error) {
	return sc.storage.CartWithItemsByCartID(ctx, cartID)
}

// CartEmpty empties a shopping cart.
func (sc *ShoppingCart) CartEmpty(ctx context.Context, cartID int64) error {
	return sc.storage.CartEmpty(ctx, cartID)
}

// LineItemAdd adds products to a shopping cart, returns items added.
func (sc *ShoppingCart) LineItemAdd(ctx context.Context, cartID int64, items []*LineItem) ([]*LineItem, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		tx  storer
		err error
	)
	tx, err = sc.storage.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("tx: %w", err)
	}

	cart, err := tx.CartWithItemsByCartID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart: %w", err)
	}

	// Sum quantity of existing products.
	for _, item := range items {
		for _, i := range cart.LineItems {
			if i.ProductID == item.ProductID {
				item.ID = i.ID
				item.Quantity += i.Quantity
			}
		}
	}

	if err := tx.LineItemsUpsert(ctx, cartID, items...); err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return items, nil
}

// LineItemRemove removes products from a shopping cart.
func (sc *ShoppingCart) LineItemRemove(ctx context.Context, cartID, itemID int64) error {
	return sc.storage.LineItemRemove(ctx, cartID, itemID)
}
