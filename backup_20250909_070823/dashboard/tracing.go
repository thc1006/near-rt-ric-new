package dashboard

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracingConfig holds the configuration for distributed tracing
type TracingConfig struct {
	ServiceName     string
	ServiceVersion  string
	JaegerEndpoint  string
	SamplingRate    float64
	Environment     string
}

// TracingManager manages distributed tracing setup and operations
type TracingManager struct {
	tracer   oteltrace.Tracer
	provider *trace.TracerProvider
	config   TracingConfig
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config TracingConfig) (*TracingManager, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for trace context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := tp.Tracer(config.ServiceName)

	return &TracingManager{
		tracer:   tracer,
		provider: tp,
		config:   config,
	}, nil
}

// Close shuts down the tracing manager
func (tm *TracingManager) Close(ctx context.Context) error {
	return tm.provider.Shutdown(ctx)
}

// StartSpan starts a new span with the given name and options
func (tm *TracingManager) StartSpan(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, spanName, opts...)
}

// StartE2Span starts a span for E2 interface operations
func (tm *TracingManager) StartE2Span(ctx context.Context, operation, nodeID string) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("e2.%s", operation),
		oteltrace.WithAttributes(
			attribute.String("interface", "E2"),
			attribute.String("operation", operation),
			attribute.String("node_id", nodeID),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)
}

// StartA1Span starts a span for A1 interface operations
func (tm *TracingManager) StartA1Span(ctx context.Context, operation, policyTypeID string) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("a1.%s", operation),
		oteltrace.WithAttributes(
			attribute.String("interface", "A1"),
			attribute.String("operation", operation),
			attribute.String("policy_type_id", policyTypeID),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)
}

// StartO1Span starts a span for O1 interface operations
func (tm *TracingManager) StartO1Span(ctx context.Context, operation, target string) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("o1.%s", operation),
		oteltrace.WithAttributes(
			attribute.String("interface", "O1"),
			attribute.String("operation", operation),
			attribute.String("target", target),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)
}

// StartSubscriptionSpan starts a span for subscription operations
func (tm *TracingManager) StartSubscriptionSpan(ctx context.Context, operation, subscriptionID, nodeID string) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("subscription.%s", operation),
		oteltrace.WithAttributes(
			attribute.String("component", "subscription"),
			attribute.String("operation", operation),
			attribute.String("subscription_id", subscriptionID),
			attribute.String("node_id", nodeID),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
	)
}

// StartHTTPSpan starts a span for HTTP operations
func (tm *TracingManager) StartHTTPSpan(ctx context.Context, method, path string) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("%s %s", method, path),
		oteltrace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", path),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(span oteltrace.Span, attrs map[string]interface{}) {
	if span == nil {
		return
	}
	
	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			attributes = append(attributes, attribute.String(k, val))
		case int:
			attributes = append(attributes, attribute.Int(k, val))
		case int64:
			attributes = append(attributes, attribute.Int64(k, val))
		case float64:
			attributes = append(attributes, attribute.Float64(k, val))
		case bool:
			attributes = append(attributes, attribute.Bool(k, val))
		default:
			attributes = append(attributes, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	
	span.SetAttributes(attributes...)
}

// AddSpanError adds error information to the span
func AddSpanError(span oteltrace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	
	span.RecordError(err)
	span.SetStatus(oteltrace.StatusError, err.Error())
}

// SetSpanSuccess marks the span as successful
func SetSpanSuccess(span oteltrace.Span) {
	if span == nil {
		return
	}
	
	span.SetStatus(oteltrace.StatusOK, "")
}

// GetTraceID extracts the trace ID from the context
func GetTraceID(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from the context
func GetSpanID(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// InjectTraceContext injects trace context into headers
func InjectTraceContext(ctx context.Context, headers map[string]string) {
	propagator := otel.GetTextMapPropagator()
	carrier := &MapCarrier{headers}
	propagator.Inject(ctx, carrier)
}

// ExtractTraceContext extracts trace context from headers
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := &MapCarrier{headers}
	return propagator.Extract(ctx, carrier)
}

// MapCarrier implements TextMapCarrier for map[string]string
type MapCarrier struct {
	data map[string]string
}

// Get returns the value associated with the passed key
func (c *MapCarrier) Get(key string) string {
	return c.data[key]
}

// Set stores the key-value pair
func (c *MapCarrier) Set(key, value string) {
	c.data[key] = value
}

// Keys lists the keys stored in this carrier
func (c *MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// TraceMiddleware creates HTTP middleware for tracing
func (tm *TracingManager) TraceMiddleware(next func(ctx context.Context, method, path string) (int, error)) func(ctx context.Context, method, path string) (int, error) {
	return func(ctx context.Context, method, path string) (int, error) {
		// Start HTTP span
		ctx, span := tm.StartHTTPSpan(ctx, method, path)
		defer span.End()
		
		// Add correlation ID to span
		correlationID := GetCorrelationID(ctx)
		if correlationID != "" {
			span.SetAttributes(attribute.String("correlation_id", correlationID))
		}
		
		// Call next handler
		statusCode, err := next(ctx, method, path)
		
		// Add response information to span
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
		
		if err != nil {
			AddSpanError(span, err)
		} else {
			SetSpanSuccess(span)
		}
		
		return statusCode, err
	}
}

// Global tracing manager instance
var GlobalTracer *TracingManager

// InitializeTracing initializes global tracing
func InitializeTracing(config TracingConfig) error {
	var err error
	GlobalTracer, err = NewTracingManager(config)
	return err
}

// CloseTracing closes global tracing
func CloseTracing(ctx context.Context) error {
	if GlobalTracer != nil {
		return GlobalTracer.Close(ctx)
	}
	return nil
}