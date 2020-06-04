package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose"
)

func TestSQLite3_BeginTx(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	st := &SQLite3{db: db}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx, err := st.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.BeginTx(ctx, nil)
	if err == nil {
		t.Error("err exp, got none")
	}
}

func TestSQLite3_Commit(t *testing.T) {
	t.Skip("🤷")
}

func TestSQLite3_CartCreate(t *testing.T) {
	c := &Cart{
		UserID: 1,
	}

	st := &SQLite3{db: connectDB(t)}
	if err := st.CartCreate(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if c.ID == 0 {
		t.Error("id not updated")
	}
}

func TestSQLite3_CartWithItemsByCartID(t *testing.T) {
	db := connectDB(t)
	c := createCartWithItems(t, db)
	st := &SQLite3{db: db}

	cart, err := st.CartWithItemsByCartID(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(c, cart) {
		t.Errorf("carts do not match\nexp: %+v\ngot: %+v", c, cart)
	}
}

func TestSQLite3_CartEmpty(t *testing.T) {
	db := connectDB(t)
	c := createCartWithItems(t, db)
	st := &SQLite3{db: db}

	if err := st.CartEmpty(context.Background(), c.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSQLite3_LineItemsUpsert(t *testing.T) {
	db := connectDB(t)
	c := createCartWithItems(t, db)
	st := &SQLite3{db: db}

	ii := []*LineItem{
		{ProductID: 9, Quantity: 5},
		{ProductID: 1, Quantity: 8},
	}

	if err := st.LineItemsUpsert(context.Background(), c.ID, ii...); err != nil {
		t.Fatal("first:", err)
	}

	if err := st.LineItemsUpsert(context.Background(), c.ID, &LineItem{ProductID: 9, Quantity: 3}); err != nil {
		t.Fatal("second:", err)
	}

	cart, err := st.CartWithItemsByCartID(context.Background(), c.ID)
	if err != nil {
		t.Fatal("cart:", err)
	}

	if l := len(cart.LineItems); l != 2 {
		t.Fatalf("cart items num exp: %d, got: %d", 2, l)
	}

	for _, i := range cart.LineItems {
		if i.ProductID == 1 && i.Quantity != 8 {
			t.Errorf("product %d quantity exp: %d got: %d", 1, 8, i.Quantity)
		} else if i.ProductID == 9 && i.Quantity != 3 {
			t.Errorf("product %d quantity exp: %d got: %d", 9, 3, i.Quantity)
		}
	}
}

func TestSQLite3_LineItemRemove(t *testing.T) {
	db := connectDB(t)
	c := createCartWithItems(t, db)
	st := &SQLite3{db: db}

	if err := st.LineItemRemove(context.Background(), c.ID, c.LineItems[0].ID); err != nil {
		t.Fatal(err)
	}
}

func connectDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	goose.SetLogger(log.New(ioutil.Discard, "", 0))

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		t.Fatal(err)
	}

	return db
}

func createCartWithItems(t *testing.T, db *sql.DB) *Cart {
	t.Helper()

	cartID := time.Now().UnixNano()

	c := &Cart{
		ID:     cartID,
		UserID: 50,
		LineItems: []*LineItem{
			{ID: time.Now().UnixNano(), CartID: cartID, ProductID: 1, Quantity: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := db.Exec(
		`INSERT INTO carts(id, user_id, created_at, updated_at) VALUES(?, ?, ?, ?)`,
		c.ID, c.UserID, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		t.Fatal("cart:", err)
	}

	_, err = db.Exec(
		`INSERT INTO line_items(id, cart_id, product_id, quantity, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?)`,
		c.LineItems[0].ID, c.LineItems[0].CartID, c.LineItems[0].ProductID, c.LineItems[0].Quantity, c.LineItems[0].CreatedAt, c.LineItems[0].UpdatedAt,
	)
	if err != nil {
		t.Fatal("item:", err)
	}

	return c
}
