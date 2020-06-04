package main

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/gojuno/minimock/v3"
)

func TestShoppingCart_CartCreate(t *testing.T) {
	c := &Cart{
		UserID: 10,
		LineItems: []*LineItem{
			{ProductID: 1, Quantity: 2},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mc := minimock.NewController(t)
	defer mc.Finish()

	tx := NewStorerMock(mc)
	tx = tx.CartCreateMock.Expect(ctx, c).Return(nil)
	tx = tx.LineItemsUpsertMock.Expect(ctx, c.ID, c.LineItems...).Return(nil)
	tx = tx.CommitMock.Expect().Return(nil)

	st := NewStorerMock(mc)
	st = st.BeginTxMock.Expect(ctx, nil).Return(tx, nil)

	sc := &ShoppingCart{storage: st}

	cart, err := sc.CartCreate(context.Background(), c.UserID, c.LineItems)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(c, cart) {
		t.Errorf("carts do not match\nexp: %+v\ngot: %+v", c, cart)
	}
}

func TestShoppingCart_CartShow(t *testing.T) {}

func TestShoppingCart_CartEmpty(t *testing.T) {}

func TestShoppingCart_LineItemAdd(t *testing.T) {
	c := &Cart{
		ID:     1,
		UserID: 10,
		LineItems: []*LineItem{
			{ID: 1, ProductID: 1, Quantity: 1},
			{ID: 2, ProductID: 2, Quantity: 2},
		},
	}

	ii := []*LineItem{
		{ProductID: 2, Quantity: 2},
		{ProductID: 3, Quantity: 1},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mc := minimock.NewController(t)
	defer mc.Finish()

	tx := NewStorerMock(mc)
	tx = tx.CartWithItemsByCartIDMock.Expect(ctx, c.ID).Return(c, nil)
	tx = tx.LineItemsUpsertMock.Set(func(_ context.Context, cartID int64, items ...*LineItem) error {
		if cartID != c.ID {
			t.Errorf("cartID exp: %d, got: %d", c.ID, cartID)
		}

		items[0].CartID = cartID

		items[1].ID = 99
		items[1].CartID = cartID

		return nil
	})
	tx = tx.CommitMock.Expect().Return(nil)

	st := NewStorerMock(mc)
	st = st.BeginTxMock.Expect(ctx, &sql.TxOptions{ReadOnly: true}).Return(tx, nil)

	sc := &ShoppingCart{storage: st}

	items, err := sc.LineItemAdd(context.Background(), c.ID, ii)
	if err != nil {
		t.Fatal(err)
	}

	exp := []*LineItem{
		{ID: 2, CartID: 1, ProductID: 2, Quantity: 4},
		{ID: 99, CartID: 1, ProductID: 3, Quantity: 1},
	}

	for i := range items {
		if !reflect.DeepEqual(items[i], exp[i]) {
			t.Errorf("items do not match\nexp: %+v\ngot: %+v", exp[i], items[i])
		}

	}
}

func TestShoppingCart_LineItemRemove(t *testing.T) {}
