package main

import (
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)

	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()
	userId := r.URL.Query().Get("userID")
	if userId == "" {
		log.Println("userID is required")
		return
	}
	connManager.Add(userId, conn)
	defer connManager.Remove(userId)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read failed: %v", err)
			break
		}
		log.Printf("received: %s", message)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)

	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userId := r.URL.Query().Get("userID")
	if userId == "" {
		log.Println("userID is required")
		return
	}
	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("packageSlug is required")
		return
	}
	connManager.Add(userId, conn)
	ctx := r.Context()
	driverService, err := grpc_clients.NewDriverServiceClient()
	defer driverService.Close()
	defer func() {
		connManager.Remove(userId)
		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userId,
			PackageSlug: packageSlug,
		})
		driverService.Close()
		log.Println("driver unregistered", userId)
	}()
	if err != nil {
		log.Fatalf("failed to create driver service client: %v", err)
	}

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userId,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("failed to register driver: %v", err)
		return
	}
	msg := contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}
	if err := connManager.SendMessage(userId, msg); err != nil {
		log.Printf("error sending message: %v", err)
		return
	}
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}
	for _, queue := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, queue)
		if err := consumer.Start(); err != nil {
			log.Printf("failed to start consumer for queue %s: %v", queue, err)
			continue
		}
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read failed: %v", err)
			break
		}
		log.Printf("received: %s", message)
	}
}
