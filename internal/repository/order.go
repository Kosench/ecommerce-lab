package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/Kosench/ecommerce-lab/platform/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id string) (*model.Order, error)
}

type pgOrderRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewOrderRepository(pool *pgxpool.Pool, logger logger.Logger) OrderRepository {
	return &pgOrderRepository{
		pool:   pool,
		logger: logger.With(zap.String("component", "repository")),
	}
}

var ErrOrderNotFound = errors.New("order not found")

func (r *pgOrderRepository) Create(ctx context.Context, order *model.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			zap.Error(err),
		)
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			r.logger.Warn("rolling back transaction",
				zap.Error(err),
			)
			tx.Rollback(ctx)
		}
	}()

	q := `INSERT INTO orders (id, user_id, status, total, created_at, updated_at) 
	      VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = tx.QueryRow(ctx, q, order.ID, order.UserID, order.Status, order.Total, order.CreatedAt, order.UpdatedAt).Scan(&order.ID)
	if err != nil {
		r.logger.Error("failed to insert order",
			zap.Error(err),
			zap.String("order_id", order.ID),
		)
		return fmt.Errorf("insert order: %w", err)
	}

	r.logger.Debug("order inserted",
		zap.String("order_id", order.ID),
	)

	for i, item := range order.Items {
		itemID := uuid.NewString()
		q = `INSERT INTO order_items (id, order_id, product_id, quantity, price) 
		      VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(ctx, q, itemID, order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			r.logger.Error("failed to insert order item",
				zap.Error(err),
				zap.String("order_id", order.ID),
				zap.Int("item_index", i),
			)
			return fmt.Errorf("insert item: %w", err)
		}
	}

	r.logger.Debug("order items inserted",
		zap.String("order_id", order.ID),
		zap.Int("items_count", len(order.Items)),
	)

	if err = tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction",
			zap.Error(err),
			zap.String("order_id", order.ID),
		)
		return fmt.Errorf("commit tx: %w", err)
	}

	r.logger.Info("order transaction committed",
		zap.String("order_id", order.ID),
	)

	return nil
}

func (r *pgOrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	q := `SELECT id, user_id, status, total, created_at, updated_at 
	      FROM orders WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)

	var order model.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		r.logger.Warn("order not found",
			zap.String("order_id", id),
		)
		return nil, ErrOrderNotFound
	}
	if err != nil {
		r.logger.Error("failed to select order",
			zap.Error(err),
			zap.String("order_id", id),
		)
		return nil, fmt.Errorf("select order: %w", err)
	}

	q = `SELECT product_id, quantity, price FROM order_items WHERE order_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		r.logger.Error("failed to query order items",
			zap.Error(err),
			zap.String("order_id", id),
		)
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			r.logger.Error("failed to scan order item",
				zap.Error(err),
				zap.String("order_id", id),
			)
			return nil, fmt.Errorf("scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating order items",
			zap.Error(err),
			zap.String("order_id", id),
		)
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	r.logger.Debug("order loaded with items",
		zap.String("order_id", order.ID),
		zap.Int("items_count", len(order.Items)),
	)

	return &order, nil
}
