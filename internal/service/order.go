package service

import (
	"context"
	"errors"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/Kosench/ecommerce-lab/internal/repository"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID string, items []model.OrderItem) (*model.Order, error)
}

type orderService struct {
	orderRepo repository.OrderRepository
}

func NewOrderService(orderRepo repository.OrderRepository) OrderService {
	return &orderService{orderRepo: orderRepo}
}

var (
	ErrInvalidRequest = errors.New("invalid request")
)

func (s *orderService) CreateOrder(ctx context.Context, userID string, items []model.OrderItem) (*model.Order, error) {
	if userID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := model.NewOrder(userID, items)
	if err != nil {
		return nil, err
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}
