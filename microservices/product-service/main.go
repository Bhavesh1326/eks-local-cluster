package main

import (
	"context"
	"encoding/json"
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
)

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

var (
	products = []Product{
		{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Coffee Mug", Description: "Ceramic coffee mug", Price: 15.99, Category: "Kitchen"},
		{ID: 3, Name: "Book", Description: "Programming guide", Price: 29.99, Category: "Books"},
		{ID: 4, Name: "Headphones", Description: "Wireless headphones", Price: 199.99, Category: "Electronics"},
	}

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

	productViewsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "product_views_total",
			Help: "Total number of product views",
		},
		[]string{"product_id", "category"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(productViewsTotal)
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
			semconv.ServiceNameKey.String("product-service"),
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

func getProducts(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("product-service")
	ctx, span := tracer.Start(r.Context(), "get-products")
	defer span.End()

	span.SetAttributes(attribute.String("operation", "get-products"))
	span.SetAttributes(attribute.Int("product.count", len(products)))

	// Add some artificial processing time
	time.Sleep(50 * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("product-service")
	ctx, span := tracer.Start(r.Context(), "get-product")
	defer span.End()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid product id"))
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("product.id", id))

	for _, product := range products {
		if product.ID == id {
			// Track product views
			productViewsTotal.WithLabelValues(strconv.Itoa(product.ID), product.Category).Inc()
			
			span.SetAttributes(attribute.String("product.name", product.Name))
			span.SetAttributes(attribute.String("product.category", product.Category))
			span.SetAttributes(attribute.Float64("product.price", product.Price))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(product)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product not found"))
	http.Error(w, "Product not found", http.StatusNotFound)
}

func getProductsByCategory(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("product-service")
	ctx, span := tracer.Start(r.Context(), "get-products-by-category")
	defer span.End()

	vars := mux.Vars(r)
	category := vars["category"]
	
	span.SetAttributes(attribute.String("product.category", category))

	var filteredProducts []Product
	for _, product := range products {
		if product.Category == category {
			filteredProducts = append(filteredProducts, product)
		}
	}

	span.SetAttributes(attribute.Int("filtered.count", len(filteredProducts)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredProducts)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":    "healthy",
		"service":   "product-service",
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
	r.HandleFunc("/products", getProducts).Methods("GET")
	r.HandleFunc("/products/{id}", getProduct).Methods("GET")
	r.HandleFunc("/products/category/{category}", getProductsByCategory).Methods("GET")
	r.HandleFunc("/health", healthCheck).Methods("GET")
	
	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Product service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
