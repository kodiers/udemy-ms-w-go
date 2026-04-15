package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = env.GetString("GRPC_ADDR", ":9092")
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer sh(ctx)
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

	service := NewService()
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	// TODO init grpc handler
	NewGrpcHandler(grpcServer, service)

	consumer := newTripEventConsumer(rabbitmq, service)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Printf("Failed to start trip event consumer: %v", err)
			cancel()
		}
	}()

	log.Printf("Starting gRPC server on: %v", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server exited with error: %v", err)
			cancel()
		}
	}()

	// wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down...")
	grpcServer.GracefulStop()
}
