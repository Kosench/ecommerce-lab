package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/google/uuid"
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

var ErrOrderNotFound = errors.New("order not found")

func (r *pgOrderRepository) Create(ctx context.Context, order *model.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := `INSERT INTO orders (id, user_id, status, total, created_at, updated_at) 
	      VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = tx.QueryRow(ctx, q, order.ID, order.UserID, order.Status, order.Total, order.CreatedAt, order.UpdatedAt).Scan(&order.ID)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for _, item := range order.Items {
		q = `INSERT INTO order_items (id, order_id, product_id, quantity, price) 
		      VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(ctx, q, uuid.NewString(), order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *pgOrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	q := `SELECT id, user_id, status, total, created_at, updated_at 
	      FROM orders WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)

	var order model.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
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
			return nil, fmt.Errorf("scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	return &order, nil
}
