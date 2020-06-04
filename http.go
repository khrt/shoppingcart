package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

type apiv1Cart struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	LineItems []apiv1LineItem `json:"line_items,omitempty"`
}

type apiv1LineItem struct {
	ID        int64 `json:"id"`
	CartID    int64 `json:"cart_id"`
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}

type service interface {
	CartCreate(ctx context.Context, userID int64, items []*LineItem) (*Cart, error)
	CartShow(ctx context.Context, cartID int64) (*Cart, error)
	CartEmpty(ctx context.Context, cartID int64) error
	LineItemAdd(ctx context.Context, cartID int64, items []*LineItem) ([]*LineItem, error)
	LineItemRemove(ctx context.Context, cartID, itemID int64) error
}

// APIv1 describes Shopping Cart REST API v1.
type APIv1 struct {
	service service
}

// NewAPIv1 instantiates APIv1.
func NewAPIv1(srv service) *chi.Mux {
	h := APIv1{service: srv}

	r := chi.NewRouter()

	r.Use(APIv1AuthMiddleware(nil))

	r.Post("/v1/cart", h.CartCreate)
	r.Get("/v1/cart/{cartID}", h.CartShow)
	r.Delete("/v1/cart/{cartID}", h.CartEmpty)

	r.Put("/v1/cart/{cartID}/item", h.LineItemAdd)
	r.Delete("/v1/cart/{cartID}/item/{itemID}", h.LineItemRemove)

	return r
}

// CartCreate creates and persists a shopping cart.
func (h *APIv1) CartCreate(w http.ResponseWriter, r *http.Request) {
	var c apiv1Cart
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "json: %s", err)
		return
	} else if c.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "user ID required")
		return
	}

	cart, err := h.service.CartCreate(r.Context(), c.UserID, h.fromAPIv1LineItem(c.LineItems))
	switch {
	case r.Context().Err() != nil:
		w.WriteHeader(http.StatusRequestTimeout)
		return
	case err != nil:
		log.Printf("CartCreate(%+v): %s", c, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(h.toAPIv1Cart(cart)); err != nil {
		log.Printf("CartCreate Encode(%+v): %s", cart, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}

// CartShow returns the details of a cart.
func (h *APIv1) CartShow(w http.ResponseWriter, r *http.Request) {
	cartID, err := h.parseInt(chi.URLParam(r, "cartID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cartID: %s", err)
		return
	}

	cart, err := h.service.CartShow(r.Context(), cartID)
	switch {
	case r.Context().Err() != nil:
		w.WriteHeader(http.StatusRequestTimeout)
		return
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		return
	case err != nil:
		log.Printf("CartShow(%d): %s", cartID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(h.toAPIv1Cart(cart)); err != nil {
		log.Printf("CartShow Encode(%+v): %s", cart, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}

// CartEmpty empties a shopping cart.
// NOTE: Empties only the cart's items, does not delete the cart itself.
func (h *APIv1) CartEmpty(w http.ResponseWriter, r *http.Request) {
	cartID, err := h.parseInt(chi.URLParam(r, "cartID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cartID: %s", err)
		return
	}

	err = h.service.CartEmpty(r.Context(), cartID)
	switch {
	case r.Context().Err() != nil:
		w.WriteHeader(http.StatusRequestTimeout)
		return
	case err != nil:
		log.Printf("CartEmpty(%d): %s", cartID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	return
}

// LineItemAdd adds products to a shopping cart.
func (h *APIv1) LineItemAdd(w http.ResponseWriter, r *http.Request) {
	cartID, err := h.parseInt(chi.URLParam(r, "cartID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cartID: %s", err)
		return
	}

	var ii []apiv1LineItem
	if err := json.NewDecoder(r.Body).Decode(&ii); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "json: %s", err)
		return
	}

	items, err := h.service.LineItemAdd(r.Context(), cartID, h.fromAPIv1LineItem(ii))
	switch {
	case r.Context().Err() != nil:
		w.WriteHeader(http.StatusRequestTimeout)
		return
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "cart does not exist")
		return
	case err != nil:
		log.Printf("LineItemAdd(%d, %+v): %s", cartID, ii, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(h.toAPIv1LineItem(items)); err != nil {
		log.Printf("LineItemAdd Encode(%+v): %s", items, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}

// LineItemRemove removes products from a shopping cart.
func (h *APIv1) LineItemRemove(w http.ResponseWriter, r *http.Request) {
	cartID, err := h.parseInt(chi.URLParam(r, "cartID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cartID: %s", err)
		return
	}

	itemID, err := h.parseInt(chi.URLParam(r, "itemID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "itemID: %s", err)
		return
	}

	err = h.service.LineItemRemove(r.Context(), cartID, itemID)
	switch {
	case r.Context().Err() != nil:
		w.WriteHeader(http.StatusRequestTimeout)
		return
	case errors.Is(err, sql.ErrNoRows):
		// Being idempotent.
		// Ignoring Cart doesn't exist error assuming the item doesn't exist as well.
	case err != nil:
		log.Printf("LineItemRemove(%d, %d): %s", cartID, itemID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	return
}

func (h *APIv1) fromAPIv1LineItem(ii []apiv1LineItem) []*LineItem {
	items := make([]*LineItem, len(ii))
	for j, i := range ii {
		items[j] = &LineItem{
			ProductID: i.ProductID,
			Quantity:  i.Quantity,
		}
	}
	return items
}

func (h *APIv1) toAPIv1Cart(cart *Cart) apiv1Cart {
	c := apiv1Cart{
		ID:     cart.ID,
		UserID: cart.UserID,
	}
	if len(cart.LineItems) > 0 {
		c.LineItems = h.toAPIv1LineItem(cart.LineItems)
	}
	return c
}

func (h *APIv1) toAPIv1LineItem(items []*LineItem) []apiv1LineItem {
	ii := make([]apiv1LineItem, len(items))
	for j, item := range items {
		ii[j] = apiv1LineItem{
			ID:        item.ID,
			CartID:    item.CartID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return ii
}

func (h *APIv1) parseInt(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q: %w", s, err)
	} else if i == 0 {
		return 0, errors.New("invalid value")
	}
	return i, nil
}

// Context Auth constants.
type ctxAuthKey uint8

const ctxAuth ctxAuthKey = 0

// APIv1AuthMiddleware returns an authentication middleware which auth users over auth service.
func APIv1AuthMiddleware(authsrv interface{}) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth() // NOTE: let's pretend it's a token
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// NOTE: sending _the token_ to the imaginary authsrv service.
			if user != "Aladdin" || pass != "OpenSesame" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			// NOTE: adding auth info the request's context.
			ctx := context.WithValue(r.Context(), ctxAuth, "info")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
