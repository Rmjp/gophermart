package main

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer connects to Jaeger and returns a shutdown function
func InitTracer(ctx context.Context, serviceName string, jaegerAddr string) (func(context.Context) error, error) {
	// 1. Create gRPC connection to Jaeger
	conn, err := grpc.DialContext(ctx, jaegerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	// 2. Create the OTLP Exporter (This sends data to Jaeger)
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	// 3. Create the Resource (Identifies this service)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 4. Create the Tracer Provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	// 5. Set Global Tracer
	otel.SetTracerProvider(tracerProvider)

	// This ensures the trace ID is passed from API Gateway -> Order Service
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Return a function to cleanly shut down
	return tracerProvider.Shutdown, nil
}
