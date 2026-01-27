package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID        string
	UserID    string
	Items     []OrderItem
	Status    OrderStatus
	Total     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     int64
}

var (
	ErrInvalidOrder    = errors.New("invalid order")
	ErrEmptyUserID     = errors.New("user_id is required")
	ErrEmptyItems      = errors.New("order must have at least one item")
	ErrInvalidProduct  = errors.New("product_id is required")
	ErrInvalidQuantity = errors.New("quantity must be positive")
	ErrInvalidPrice    = errors.New("price must be positive")
)

func NewOrder(userID string, items []OrderItem) (*Order, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	for i, item := range items {
		if item.ProductID == "" {
			return nil, fmt.Errorf("%w: item[%d]", ErrInvalidProduct, i)
		}
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: item[%d]", ErrInvalidQuantity, i)
		}
		if item.Price <= 0 {
			return nil, fmt.Errorf("%w: item[%d]", ErrInvalidPrice, i)
		}
	}

	var total int64
	for _, item := range items {
		total += int64(item.Quantity) * item.Price
	}

	now := time.Now()
	return &Order{
		ID:        uuid.NewString(),
		UserID:    userID,
		Items:     items,
		Status:    StatusPending,
		Total:     total,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
