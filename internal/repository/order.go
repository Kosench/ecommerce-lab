package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id string) (*model.Order, error)
}

type pgOrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &pgOrderRepository{pool: pool}
}

func (r *pgOrderRepository) Create(ctx context.Context, order *model.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Generate ID in the database if not provided
	if order.ID == "" {
		q := `INSERT INTO orders (user_id, status, total, created_at, updated_at)
		      VALUES ($1, $2, $3, $4, $5) RETURNING id`
		err = tx.QueryRow(ctx, q, order.UserID, order.Status, order.Total, order.CreatedAt, order.UpdatedAt).Scan(&order.ID)
		if err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
	} else {
		q := `INSERT INTO orders (id, user_id, status, total, created_at, updated_at)
		      VALUES ($1, $2, $3, $4, $5, $6)`
		_, err = tx.Exec(ctx, q, order.ID, order.UserID, order.Status, order.Total, order.CreatedAt, order.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert order with ID: %w", err)
		}
	}

	// Insert order items
	for _, item := range order.Items {
		q := `INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(ctx, q, order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *pgOrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	q := `SELECT id, user_id, status, total, created_at, updated_at
	      FROM orders WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)

	var order model.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("order not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("select order: %w", err)
	}

	q = `SELECT product_id, quantity, price FROM order_items WHERE order_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("scan item %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}
