package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/Kosench/ecommerce-lab/internal/model"
	"github.com/Kosench/ecommerce-lab/internal/repository"
	"github.com/Kosench/ecommerce-lab/internal/service"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

type createOrderRequest struct {
	UserID string       `json:"user_id"`
	Items  []createItem `json:"items"`
}

type createItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"`
}

type createOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, `{"error": "user_id is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, `{"error": "items are required"}`, http.StatusBadRequest)
		return
	}

	if !isValidUUID(req.UserID) {
		http.Error(w, `{"error": "user_id must be a valid UUID"}`, http.StatusBadRequest)
		return
	}

	items := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		if !isValidUUID(item.ProductID) {
			http.Error(w, fmt.Sprintf(`{"error": "item[%d].product_id must be a valid UUID"}`, i), http.StatusBadRequest)
			return
		}
		if item.Quantity <= 0 {
			http.Error(w, fmt.Sprintf(`{"error": "item[%d].quantity must be positive"}`, i), http.StatusBadRequest)
			return
		}
		if item.Price <= 0 {
			http.Error(w, fmt.Sprintf(`{"error": "item[%d].price must be positive"}`, i), http.StatusBadRequest)
			return
		}

		items[i] = model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	order, err := h.orderService.CreateOrder(r.Context(), req.UserID, items)
	if err != nil {
		log.Printf("ERROR creating order: %v", err)

		if errors.Is(err, service.ErrInvalidRequest) ||
			errors.Is(err, model.ErrEmptyUserID) ||
			errors.Is(err, model.ErrEmptyItems) ||
			errors.Is(err, model.ErrInvalidProduct) ||
			errors.Is(err, model.ErrInvalidQuantity) ||
			errors.Is(err, model.ErrInvalidPrice) {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		if errors.Is(err, repository.ErrOrderNotFound) {
			http.Error(w, `{"error": "order not found"}`, http.StatusNotFound)
			return
		}

		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := createOrderResponse{
		ID:     order.ID,
		Status: string(order.Status),
		Total:  order.Total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
