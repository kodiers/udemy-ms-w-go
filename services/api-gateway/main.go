package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"
	"time"

	"ride-sharing/shared/env"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMQURL = env.GetString("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	log.Println("Starting API Gateway")
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()
	mux := http.NewServeMux()
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()
	mux.Handle("POST /trip/preview", tracing.WrapHandlerFunc(enableCors(handleTripPreview), "handleTripPreview"))
	mux.Handle("POST /trip/start", tracing.WrapHandlerFunc(enableCors(handleTripCreate), "handleTripCreate"))
	mux.Handle("/ws/drivers", tracing.WrapHandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handleDriversWebSocket(writer, request, rabbitmq)
	}, "handleDriversWebSocket"))
	mux.Handle("/ws/riders", tracing.WrapHandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handleRidersWebSocket(writer, request, rabbitmq)
	}, "handleRidersWebSocket"))
	mux.Handle("/webhook/strip", tracing.WrapHandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handleStripeWebhook(writer, request, rabbitmq)
	}, "handleStripeWebhook"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Starting server on: %v", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Server exited with error: %v", err)
	case sig := <-shutdown:
		log.Printf("Received signal to shutdown: %v", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server stop gracefully failed: %v", err)
			server.Close()
		}

	}
}
