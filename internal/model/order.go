package model

import (
	"errors"
	"fmt"
	"time"
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
	UpdateAt  time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     int64
}

func NewOrder(userID string, items []OrderItem) (*Order, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if len(items) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	for i, item := range items {
		if item.ProductID == "" {
			return nil, fmt.Errorf("item[%d]: product_id is required", i)
		}
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("item[%d]: quantity must be positive", i)
		}
		if item.Price <= 0 {
			return nil, fmt.Errorf("item[%d]: price must be positive", i)
		}
	}

	var total int64
	for _, item := range items {
		total += int64(item.Quantity) * item.Price
	}

	now := time.Now()
	return &Order{
		ID:        "",
		UserID:    userID,
		Items:     items,
		Status:    StatusPending,
		Total:     total,
		CreatedAt: now,
		UpdateAt:  now,
	}, nil
}
