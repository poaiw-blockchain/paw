// Package telemetry provides OpenTelemetry tracing and metrics instrumentation
// for the PAW blockchain application. It configures distributed tracing with Jaeger,
// metrics collection with Prometheus, and provides helpers for instrumenting
// blockchain operations.
package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	serviceName    = "paw-blockchain"
	serviceVersion = "1.0.0"
)

// Config holds the configuration for telemetry
type Config struct {
	// Tracing configuration
	Enabled        bool
	JaegerEndpoint string
	SampleRate     float64
	Environment    string
	ChainID        string

	// Metrics configuration
	PrometheusEnabled bool
	MetricsPort       string
}

// Provider manages OpenTelemetry tracing and metrics
type Provider struct {
	tracerProvider *tracesdk.TracerProvider
	meterProvider  *metricsdk.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	config         Config
}

// NewProvider initializes a new telemetry provider with tracing and metrics
func NewProvider(cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{config: cfg}, nil
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid telemetry config: %w", err)
	}

	// Create resource with service information
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	provider := &Provider{config: cfg}

	// Initialize tracing
	if err := provider.initTracing(res); err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics
	if cfg.PrometheusEnabled {
		if err := provider.initMetrics(res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	return provider, nil
}

// validateConfig validates the telemetry configuration
func validateConfig(cfg Config) error {
	if cfg.JaegerEndpoint == "" {
		return fmt.Errorf("jaeger endpoint is required")
	}

	if _, err := url.Parse(cfg.JaegerEndpoint); err != nil {
		return fmt.Errorf("invalid jaeger endpoint: %w", err)
	}

	if cfg.SampleRate < 0 || cfg.SampleRate > 1 {
		return fmt.Errorf("sample rate must be between 0 and 1")
	}

	return nil
}

// newResource creates an OpenTelemetry resource with service information
func newResource(cfg Config) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			attribute.String("environment", cfg.Environment),
			attribute.String("chain.id", cfg.ChainID),
		),
	)
}

// initTracing sets up OTLP/HTTP tracing with Jaeger
func (p *Provider) initTracing(res *resource.Resource) error {
	// Parse and clean endpoint
	endpoint := strings.TrimPrefix(p.config.JaegerEndpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Create OTLP HTTP exporter
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Use HTTP not HTTPS for local Jaeger
		otlptracehttp.WithURLPath("/v1/traces"),
	)

	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create trace provider with sampling
	sampler := tracesdk.ParentBased(
		tracesdk.TraceIDRatioBased(p.config.SampleRate),
	)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter,
			tracesdk.WithMaxExportBatchSize(512),
			tracesdk.WithMaxQueueSize(2048),
			tracesdk.WithBatchTimeout(5*time.Second),
		),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	p.tracerProvider = tp
	p.tracer = tp.Tracer(serviceName)

	return nil
}

// initMetrics sets up Prometheus metrics
func (p *Provider) initMetrics(res *resource.Resource) error {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithResource(res),
		metricsdk.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	p.meterProvider = mp
	p.meter = mp.Meter(serviceName)

	return nil
}

// Shutdown gracefully shuts down the telemetry provider
func (p *Provider) Shutdown(ctx context.Context) error {
	var err error

	if p.tracerProvider != nil {
		if shutdownErr := p.tracerProvider.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr)
		}
	}

	if p.meterProvider != nil {
		if shutdownErr := p.meterProvider.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown meter provider: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr)
			}
		}
	}

	return err
}

// Tracer returns the OpenTelemetry tracer
func (p *Provider) Tracer() trace.Tracer {
	if p.tracer == nil {
		return otel.Tracer(serviceName)
	}
	return p.tracer
}

// Meter returns the OpenTelemetry meter
func (p *Provider) Meter() metric.Meter {
	if p.meter == nil {
		return otel.Meter(serviceName)
	}
	return p.meter
}

// TracingHelpers provides helper functions for instrumenting blockchain operations

// StartTxSpan starts a new span for transaction execution
func StartTxSpan(ctx context.Context, tx sdk.Tx, height int64) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, "transaction.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.Int64("block.height", height),
			attribute.Int("tx.msg.count", len(tx.GetMsgs())),
		),
	)
	return ctx, span
}

// StartModuleSpan starts a new span for module execution
func StartModuleSpan(ctx context.Context, moduleName string, operation string) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("module.%s.%s", moduleName, operation),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("module.name", moduleName),
			attribute.String("module.operation", operation),
		),
	)
	return ctx, span
}

// StartBlockSpan starts a new span for block processing
func StartBlockSpan(ctx context.Context, height int64, proposer string) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, "block.process",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.Int64("block.height", height),
			attribute.String("block.proposer", proposer),
		),
	)
	return ctx, span
}

// StartIBCSpan starts a new span for IBC packet handling
func StartIBCSpan(ctx context.Context, channel, port, sequence string) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, "ibc.packet",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("ibc.channel", channel),
			attribute.String("ibc.port", port),
			attribute.String("ibc.sequence", sequence),
		),
	)
	return ctx, span
}

// RecordError records an error on the current span
func RecordError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanStatus sets the status of a span
func SetSpanStatus(span trace.Span, success bool, message string) {
	if span == nil {
		return
	}

	if success {
		span.SetStatus(codes.Ok, message)
	} else {
		span.SetStatus(codes.Error, message)
	}
}

// AddSpanAttributes adds attributes to a span
func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to a span
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// HealthCheck verifies that telemetry is properly initialized
// Returns nil if healthy, error otherwise
func (p *Provider) HealthCheck() error {
	if !p.config.Enabled {
		return nil // Metrics disabled, nothing to check
	}

	// Check tracer provider is initialized
	if p.tracerProvider == nil {
		return fmt.Errorf("tracer provider not initialized")
	}

	// Check meter provider is initialized if Prometheus is enabled
	if p.config.PrometheusEnabled && p.meterProvider == nil {
		return fmt.Errorf("meter provider not initialized but Prometheus is enabled")
	}

	// Verify we can create spans (tracer is functional)
	if p.tracer == nil {
		return fmt.Errorf("tracer not initialized")
	}

	// Verify we can access meters if Prometheus is enabled
	if p.config.PrometheusEnabled && p.meter == nil {
		return fmt.Errorf("meter not initialized but Prometheus is enabled")
	}

	return nil
}
