package service

import (
	"context"
	"errors"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/Kosench/ecommerce-lab/internal/repository"
	"github.com/Kosench/ecommerce-lab/platform/logger"
	"go.uber.org/zap"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID string, items []model.OrderItem) (*model.Order, error)
}

type orderService struct {
	orderRepo repository.OrderRepository
	logger    logger.Logger
}

func NewOrderService(orderRepo repository.OrderRepository, logger logger.Logger) OrderService {
	return &orderService{
		orderRepo: orderRepo,
		logger:    logger.With(zap.String("component", "service"))}
}

var (
	ErrInvalidRequest = errors.New("invalid request")
)

func (s *orderService) CreateOrder(ctx context.Context, userID string, items []model.OrderItem) (*model.Order, error) {
	if userID == "" {
		s.logger.Warn("empty user_id")
		return nil, ErrInvalidRequest
	}

	order, err := model.NewOrder(userID, items)
	if err != nil {
		s.logger.Warn("invalid order model",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return nil, err
	}

	s.logger.Debug("creating order in repository",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.Int64("total", order.Total),
	)

	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.Error("failed to save order to repository",
			zap.Error(err),
			zap.String("order_id", order.ID),
		)
		return nil, err
	}

	s.logger.Info("order created",
		zap.String("order_id", order.ID),
	)

	return order, nil
}
