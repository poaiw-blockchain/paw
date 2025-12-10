package app

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"net/url"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	serviceName = "paw-blockchain"
	environment = "testnet"
)

// TelemetryConfig holds the configuration for telemetry
type TelemetryConfig struct {
	Enabled           bool
	JaegerEndpoint    string
	PrometheusEnabled bool
	MetricsPort       string
	SampleRate        float64
}

// Telemetry manages OpenTelemetry tracing and metrics
type Telemetry struct {
	tracer       *trace.TracerProvider
	meter        metric.Meter
	config       TelemetryConfig
	shutdownFunc func(context.Context) error
}

// InitTelemetry initializes OpenTelemetry tracing and metrics
func InitTelemetry(cfg TelemetryConfig) (*Telemetry, error) {
	if !cfg.Enabled {
		return &Telemetry{config: cfg}, nil
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", environment),
			attribute.String("chain.id", "paw-testnet"),
		),
	)
	if err != nil {
		return nil, err
	}

	tel := &Telemetry{config: cfg}

	// Initialize tracing
	if err := tel.initTracing(res); err != nil {
		return nil, err
	}

	// Initialize metrics
	if err := tel.initMetrics(res); err != nil {
		return nil, err
	}

	return tel, nil
}

// initTracing sets up OTLP/HTTP tracing
func (t *Telemetry) initTracing(res *resource.Resource) error {
	// Validate endpoint
	if _, err := url.Parse(t.config.JaegerEndpoint); err != nil {
		return err
	}

	endpoint := strings.TrimPrefix(t.config.JaegerEndpoint, "http://")
	exp, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return err
	}

	// Create trace provider with sampling
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(trace.ParentBased(
			trace.TraceIDRatioBased(t.config.SampleRate),
		)),
	)

	otel.SetTracerProvider(tp)
	t.tracer = tp
	t.shutdownFunc = tp.Shutdown

	return nil
}

// initMetrics sets up Prometheus metrics
func (t *Telemetry) initMetrics(res *resource.Resource) error {
	if !t.config.PrometheusEnabled {
		return nil
	}

	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}

	// Create meter provider with custom views for histograms
	provider := metricsdk.NewMeterProvider(
		metricsdk.WithResource(res),
		metricsdk.WithReader(exporter),
	)

	otel.SetMeterProvider(provider)
	t.meter = provider.Meter(serviceName)

	return nil
}

// Shutdown gracefully shuts down telemetry
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.shutdownFunc != nil {
		return t.shutdownFunc(ctx)
	}
	return nil
}

// TelemetryMiddleware wraps transactions with tracing
type TelemetryMiddleware struct {
	tracer metric.Meter

	// Metrics
	txCounter   metric.Int64Counter
	txDuration  metric.Float64Histogram
	txGasUsed   metric.Int64Histogram
	blockHeight metric.Int64Gauge
	moduleExec  metric.Float64Histogram
}

// NewTelemetryMiddleware creates a new telemetry middleware
func NewTelemetryMiddleware(meter metric.Meter) (*TelemetryMiddleware, error) {
	txCounter, err := meter.Int64Counter(
		"cosmos.tx.total",
		metric.WithDescription("Total number of transactions"),
		metric.WithUnit("{transaction}"),
	)
	if err != nil {
		return nil, err
	}

	txDuration, err := meter.Float64Histogram(
		"cosmos.tx.processing_time",
		metric.WithDescription("Transaction processing time"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	txGasUsed, err := meter.Int64Histogram(
		"cosmos.tx.gas_used",
		metric.WithDescription("Gas used by transaction"),
		metric.WithUnit("{gas}"),
	)
	if err != nil {
		return nil, err
	}

	blockHeight, err := meter.Int64Gauge(
		"cosmos.block.height",
		metric.WithDescription("Current block height"),
		metric.WithUnit("{block}"),
	)
	if err != nil {
		return nil, err
	}

	moduleExec, err := meter.Float64Histogram(
		"cosmos.module.execution_time",
		metric.WithDescription("Module execution time"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &TelemetryMiddleware{
		tracer:      meter,
		txCounter:   txCounter,
		txDuration:  txDuration,
		txGasUsed:   txGasUsed,
		blockHeight: blockHeight,
		moduleExec:  moduleExec,
	}, nil
}

// RecordTransaction records transaction metrics
func (tm *TelemetryMiddleware) RecordTransaction(
	ctx context.Context,
	txType string,
	duration time.Duration,
	gasUsed int64,
	success bool,
) {
	status := "success"
	if !success {
		status = "failed"
	}

	attrs := []attribute.KeyValue{
		attribute.String("tx.type", txType),
		attribute.String("tx.status", status),
	}

	tm.txCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	tm.txDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
	tm.txGasUsed.Record(ctx, gasUsed, metric.WithAttributes(attrs...))
}

// RecordBlockHeight records the current block height
func (tm *TelemetryMiddleware) RecordBlockHeight(ctx context.Context, height int64) {
	tm.blockHeight.Record(ctx, height)
}

// RecordModuleExecution records module execution time
func (tm *TelemetryMiddleware) RecordModuleExecution(
	ctx context.Context,
	moduleName string,
	duration time.Duration,
) {
	attrs := []attribute.KeyValue{
		attribute.String("module.name", moduleName),
	}
	tm.moduleExec.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// TraceTxExecution creates a traced context for transaction execution
func TraceTxExecution(ctx context.Context, tx sdk.Tx, height int64) (context.Context, func()) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, "transaction.execute")

	span.SetAttributes(
		attribute.Int64("block.height", height),
		attribute.Int("tx.msg.count", len(tx.GetMsgs())),
	)

	return ctx, func() { span.End() }
}

// TraceModuleExecution creates a traced context for module execution
func TraceModuleExecution(ctx context.Context, moduleName string) (context.Context, func()) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, "module.execute")
	span.SetAttributes(attribute.String("module.name", moduleName))
	return ctx, func() { span.End() }
}
