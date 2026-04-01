package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = env.GetString("GRPC_ADDR", ":9092")
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
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
	grpcServer := grpcserver.NewServer()
	// TODO init grpc handler
	NewGrpcHandler(grpcServer, service)

	consumer := newTripEventConsumer(rabbitmq)
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
