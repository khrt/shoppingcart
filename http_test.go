package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/gojuno/minimock/v3"
)

func TestAPIv1_CartCreate(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		c := Cart{
			UserID: 15,
			LineItems: []*LineItem{
				{ProductID: 30, Quantity: 2},
			},
		}

		uri := "/v1/cart"
		r := httptest.NewRequest(http.MethodPost, uri, bytes.NewBufferString(`{"user_id":15,"line_items":[{"id":20,"cart_id":10,"product_id":30,"quantity":2}]}`))

		mc := minimock.NewController(t)
		defer mc.Finish()

		s := NewServiceMock(mc)
		s = s.CartCreateMock.Expect(r.Context(), c.UserID, c.LineItems).Return(&c, nil)

		w := httptest.NewRecorder()
		(&APIv1{service: s}).CartCreate(w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("code exp: %d, got: %d", http.StatusCreated, w.Code)
		}

		var cart apiv1Cart
		if err := json.NewDecoder(w.Body).Decode(&cart); err != nil {
			t.Fatal(err)
		}

		exp := apiv1Cart{
			UserID: c.UserID,
			LineItems: []apiv1LineItem{
				{CartID: c.LineItems[0].CartID, ProductID: c.LineItems[0].ProductID, Quantity: c.LineItems[0].Quantity},
			},
		}

		if !reflect.DeepEqual(exp, cart) {
			t.Errorf("carts do not match\nexp: %+v\ngot: %+v\n", exp, cart)
		}
	})

	t.Run("context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		uri := "/v1/cart"
		r, _ := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBufferString(`{"user_id":15}`))

		time.Sleep(100 * time.Millisecond)

		mc := minimock.NewController(t)
		defer mc.Finish()

		s := NewServiceMock(mc)
		s = s.CartCreateMock.Expect(r.Context(), 15, []*LineItem{}).Return(nil, nil)

		w := httptest.NewRecorder()
		(&APIv1{service: s}).CartCreate(w, r)

		if w.Code != http.StatusRequestTimeout {
			t.Errorf("code exp: %d, got: %d", http.StatusRequestTimeout, w.Code)
		}
	})
}

func TestAPIv1_CartShow(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c := Cart{
			ID:     10,
			UserID: 15,
			LineItems: []*LineItem{
				{ID: 20, CartID: 10, ProductID: 30, Quantity: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			CreatedAt: time.Now().Add(-10 * time.Minute),
			UpdatedAt: time.Now().Add(-10 * time.Minute),
		}

		uri := fmt.Sprintf("/v1/cart/%d", c.ID)
		r := httptest.NewRequest(http.MethodGet, uri, nil)
		r = r.WithContext(chiRouteContext(t, "/v1/cart/{cartID}", uri))

		mc := minimock.NewController(t)
		defer mc.Finish()

		s := NewServiceMock(mc)
		s = s.CartShowMock.Expect(r.Context(), c.ID).Return(&c, nil)

		w := httptest.NewRecorder()
		(&APIv1{service: s}).CartShow(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("code exp: %d, got: %d", http.StatusOK, w.Code)
		}

		var cart apiv1Cart
		if err := json.NewDecoder(w.Body).Decode(&cart); err != nil {
			t.Fatal(err)
		}

		exp := apiv1Cart{
			ID:     c.ID,
			UserID: c.UserID,
			LineItems: []apiv1LineItem{
				{ID: c.LineItems[0].ID, CartID: c.LineItems[0].CartID, ProductID: c.LineItems[0].ProductID, Quantity: c.LineItems[0].Quantity},
			},
		}

		if !reflect.DeepEqual(exp, cart) {
			t.Errorf("carts do not match\nexp: %+v\ngot: %+v\n", exp, cart)
		}
	})

	tests := []struct {
		name string
		err  error
		code int
	}{
		{"no rows", sql.ErrNoRows, http.StatusNotFound},
		{"any error", errors.New("any"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "/v1/cart/100"
			r := httptest.NewRequest(http.MethodGet, uri, nil)
			r = r.WithContext(chiRouteContext(t, "/v1/cart/{cartID}", uri))

			mc := minimock.NewController(t)
			defer mc.Finish()

			s := NewServiceMock(mc)
			s = s.CartShowMock.Expect(r.Context(), 100).Return(nil, tt.err)

			w := httptest.NewRecorder()
			(&APIv1{service: s}).CartShow(w, r)

			if w.Code != tt.code {
				t.Errorf("code exp: %d, got: %d", tt.code, w.Code)
			}
		})
	}
}

func TestAPIv1_CartEmpty(t *testing.T) {
	var cartID int64 = 10
	uri := fmt.Sprintf("/v1/cart/%d", cartID)
	r := httptest.NewRequest(http.MethodDelete, uri, nil)
	r = r.WithContext(chiRouteContext(t, "/v1/cart/{cartID}", uri))

	mc := minimock.NewController(t)
	defer mc.Finish()

	s := NewServiceMock(mc)
	s = s.CartEmptyMock.Expect(r.Context(), cartID).Return(nil)

	w := httptest.NewRecorder()
	(&APIv1{service: s}).CartEmpty(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("code exp: %d, got: %d", http.StatusNoContent, w.Code)
	}

	// TODO tests
}

func TestAPIv1_LineItemAdd(t *testing.T) {
	var cartID int64 = 40
	ii := []*LineItem{
		{ProductID: 30, Quantity: 2},
	}

	uri := fmt.Sprintf("/v1/cart/%d/item", cartID)
	r := httptest.NewRequest(http.MethodPost, uri, bytes.NewBufferString(`[{"id":20,"cart_id":10,"product_id":30,"quantity":2}]`))
	r = r.WithContext(chiRouteContext(t, "/v1/cart/{cartID}/item", uri))

	mc := minimock.NewController(t)
	defer mc.Finish()

	s := NewServiceMock(mc)
	s = s.LineItemAddMock.Expect(r.Context(), cartID, ii).Return(ii, nil)

	w := httptest.NewRecorder()
	(&APIv1{service: s}).LineItemAdd(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("code exp: %d, got: %d", http.StatusCreated, w.Code)
	}

	var items []apiv1LineItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}

	exp := []apiv1LineItem{
		{ProductID: ii[0].ProductID, Quantity: ii[0].Quantity},
	}

	if !reflect.DeepEqual(exp, items) {
		t.Errorf("carts do not match\nexp: %+v\ngot: %+v\n", exp, items)
	}

	// TODO tests
}

func TestAPIv1_LineItemRemove(t *testing.T) {
	var (
		cartID int64 = 10
		itemID int64 = 20
	)

	uri := fmt.Sprintf("/v1/cart/%d/item/%d", cartID, itemID)
	r := httptest.NewRequest(http.MethodDelete, uri, nil)
	r = r.WithContext(chiRouteContext(t, "/v1/cart/{cartID}/item/{itemID}", uri))

	mc := minimock.NewController(t)
	defer mc.Finish()

	s := NewServiceMock(mc)
	s = s.LineItemRemoveMock.Expect(r.Context(), cartID, itemID).Return(nil)

	w := httptest.NewRecorder()
	(&APIv1{service: s}).LineItemRemove(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("code exp: %d, got: %d", http.StatusNoContent, w.Code)
	}

	// TODO tests
}

func chiRouteContext(t *testing.T, pattern string, uri string) context.Context {
	t.Helper()

	rctx := chi.NewRouteContext()
	rt := chi.NewRouter()
	rt.MethodFunc(http.MethodGet, pattern, http.NotFound)
	if !rt.Match(rctx, http.MethodGet, uri) {
		t.Fatalf("url did not match. pattern:%s path:%s", pattern, uri)
	}

	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}
