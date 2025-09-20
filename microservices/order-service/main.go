package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type Order struct {
	ID       int     `json:"id"`
	UserID   int     `json:"user_id"`
	Products []int   `json:"product_ids"`
	Total    float64 `json:"total"`
	Status   string  `json:"status"`
	Created  string  `json:"created"`
}

type CreateOrderRequest struct {
	UserID   int   `json:"user_id"`
	Products []int `json:"product_ids"`
}

var (
	orders = []Order{
		{ID: 1, UserID: 1, Products: []int{1, 2}, Total: 1015.98, Status: "completed", Created: "2024-01-15T10:30:00Z"},
		{ID: 2, UserID: 2, Products: []int{3}, Total: 29.99, Status: "pending", Created: "2024-01-15T11:15:00Z"},
	}
	nextOrderID = 3

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	ordersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_total",
			Help: "Total number of orders created",
		},
		[]string{"status"},
	)

	orderValue = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_value_dollars",
			Help:    "Value of orders in dollars",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000},
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(ordersTotal)
	prometheus.MustRegister(orderValue)
}

func initTracer() (*tracesdk.TracerProvider, error) {
	jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerEndpoint = "http://jaeger-collector.observability.svc.cluster.local:14268/api/traces"
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("order-service"),
			semconv.ServiceVersionKey.String("v1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.statusCode)).Inc()
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func callUserService(ctx context.Context, userID int) error {
	tracer := otel.Tracer("order-service")
	_, span := tracer.Start(ctx, "call-user-service")
	defer span.End()

	span.SetAttributes(attribute.String("external.service", "user-service"))
	span.SetAttributes(attribute.Int("user.id", userID))

	// Simulate API call to user service
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://user-service.default.svc.cluster.local:8080"
	}

	url := fmt.Sprintf("%s/users/%d", userServiceURL, userID)
	span.SetAttributes(attribute.String("http.url", url))

	// Simulate network delay
	time.Sleep(25 * time.Millisecond)

	// For demo purposes, assume user exists if ID > 0
	if userID <= 0 {
		span.SetAttributes(attribute.String("error", "invalid user"))
		return fmt.Errorf("user not found")
	}

	return nil
}

func callProductService(ctx context.Context, productID int) (float64, error) {
	tracer := otel.Tracer("order-service")
	_, span := tracer.Start(ctx, "call-product-service")
	defer span.End()

	span.SetAttributes(attribute.String("external.service", "product-service"))
	span.SetAttributes(attribute.Int("product.id", productID))

	// Simulate API call to product service
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "http://product-service.default.svc.cluster.local:8080"
	}

	url := fmt.Sprintf("%s/products/%d", productServiceURL, productID)
	span.SetAttributes(attribute.String("http.url", url))

	// Simulate network delay
	time.Sleep(30 * time.Millisecond)

	// Mock product prices for demo
	prices := map[int]float64{
		1: 999.99,
		2: 15.99,
		3: 29.99,
		4: 199.99,
	}

	if price, exists := prices[productID]; exists {
		span.SetAttributes(attribute.Float64("product.price", price))
		return price, nil
	}

	span.SetAttributes(attribute.String("error", "product not found"))
	return 0, fmt.Errorf("product not found")
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(r.Context(), "get-orders")
	defer span.End()

	span.SetAttributes(attribute.String("operation", "get-orders"))
	span.SetAttributes(attribute.Int("order.count", len(orders)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(r.Context(), "get-order")
	defer span.End()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid order id"))
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("order.id", id))

	for _, order := range orders {
		if order.ID == id {
			span.SetAttributes(attribute.String("order.status", order.Status))
			span.SetAttributes(attribute.Float64("order.total", order.Total))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(order)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "order not found"))
	http.Error(w, "Order not found", http.StatusNotFound)
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(r.Context(), "create-order")
	defer span.End()

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetAttributes(attribute.String("error", "invalid request body"))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("user.id", req.UserID))
	span.SetAttributes(attribute.IntSlice("product.ids", req.Products))

	// Validate user exists
	if err := callUserService(ctx, req.UserID); err != nil {
		span.SetAttributes(attribute.String("error", "user validation failed"))
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// Calculate total by calling product service for each product
	var total float64
	for _, productID := range req.Products {
		price, err := callProductService(ctx, productID)
		if err != nil {
			span.SetAttributes(attribute.String("error", "product validation failed"))
			http.Error(w, fmt.Sprintf("Product %d not found", productID), http.StatusBadRequest)
			return
		}
		total += price
	}

	// Create order
	order := Order{
		ID:       nextOrderID,
		UserID:   req.UserID,
		Products: req.Products,
		Total:    total,
		Status:   "pending",
		Created:  time.Now().Format(time.RFC3339),
	}
	
	orders = append(orders, order)
	nextOrderID++

	// Record metrics
	ordersTotal.WithLabelValues(order.Status).Inc()
	orderValue.WithLabelValues(order.Status).Observe(order.Total)

	span.SetAttributes(attribute.Int("order.id", order.ID))
	span.SetAttributes(attribute.Float64("order.total", order.Total))
	span.SetAttributes(attribute.String("order.status", order.Status))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":    "healthy",
		"service":   "order-service",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Initialize tracing
	tp, err := initTracer()
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	r := mux.NewRouter()
	
	// Apply Prometheus middleware
	r.Use(prometheusMiddleware)

	// API routes
	r.HandleFunc("/orders", getOrders).Methods("GET")
	r.HandleFunc("/orders/{id}", getOrder).Methods("GET")
	r.HandleFunc("/orders", createOrder).Methods("POST")
	r.HandleFunc("/health", healthCheck).Methods("GET")
	
	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Order service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
