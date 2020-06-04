package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type db interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// SQLite3 holds functions to mutate objects state in the DB.
type SQLite3 struct {
	db db
}

func (s *SQLite3) BeginTx(ctx context.Context, opts *sql.TxOptions) (storer, error) { // NOTE: consequence of storer
	db, ok := s.db.(*sql.DB)
	if !ok {
		return nil, errors.New("can not begin transaction while in a transaction")
	}

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &SQLite3{db: tx}, nil
}

func (s *SQLite3) Commit() error {
	tx, ok := s.db.(*sql.Tx)
	if !ok {
		return errors.New("not a transaction")
	}
	return tx.Commit()
}

func (s *SQLite3) CartCreate(ctx context.Context, cart *Cart) error {
	tm := time.Now().UTC()

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO carts(user_id, created_at, updated_at) VALUES(?, ?, ?)`,
		cart.UserID, tm, tm,
	)
	if err != nil {
		return err
	}

	if cart.ID, err = res.LastInsertId(); err != nil {
		return err
	}

	cart.CreatedAt, cart.UpdatedAt = tm, tm
	return nil
}

func (s *SQLite3) CartWithItemsByCartID(ctx context.Context, cartID int64) (*Cart, error) {
	c := &Cart{}
	err := s.db.QueryRowContext(
		ctx,
		`SELECT user_id, created_at, updated_at
		FROM carts
		WHERE id = ?`,
		cartID,
	).Scan(
		&c.UserID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("cart query: %w", err)
	}

	c.ID = cartID

	var ii []*LineItem
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, product_id, quantity, created_at, updated_at
		FROM line_items
		WHERE cart_id = ?`,
		cartID,
	)
	if err != nil {
		return nil, fmt.Errorf("item query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		i := &LineItem{}
		err := rows.Scan(
			&i.ID,
			&i.ProductID,
			&i.Quantity,
			&i.CreatedAt,
			&i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("item scan: %w", err)
		}

		i.CartID = cartID
		ii = append(ii, i)
	}

	c.LineItems = ii
	return c, nil
}

func (s *SQLite3) CartEmpty(ctx context.Context, cartID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM line_items WHERE cart_id = ?`,
		cartID,
	)
	return err
}

func (s *SQLite3) LineItemsUpsert(ctx context.Context, cartID int64, items ...*LineItem) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, item := range items {
		if item == nil {
			continue
		}

		tm := time.Now().UTC()

		res, err := s.db.ExecContext(
			ctx,
			`INSERT INTO line_items(cart_id, product_id, quantity, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?)
			ON CONFLICT(cart_id, product_id) DO UPDATE SET quantity = ?, updated_at = ?`,
			cartID, item.ProductID, item.Quantity, tm, tm,
			item.Quantity, tm,
		)
		if err != nil {
			return fmt.Errorf("exec %d: %w", item.ProductID, err)
		}

		item.UpdatedAt = time.Now().UTC()

		if item.ID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("id %d: %w", item.ProductID, err)
		}
	}

	return nil
}

func (s *SQLite3) LineItemRemove(ctx context.Context, cartID, itemID int64) error {
	fmt.Println(cartID, itemID)
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM line_items WHERE cart_id = ? AND id = ?`,
		cartID, itemID,
	)
	return err
}
