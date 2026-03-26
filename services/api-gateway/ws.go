package main

import (
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/proto/driver"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
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
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read failed: %v", err)
			break
		}
		log.Printf("received: %s", message)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
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
	ctx := r.Context()
	driverService, err := grpc_clients.NewDriverServiceClient()
	defer driverService.Close()
	defer func() {
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
		Type: "driver.cmd.register",
		Data: driverData.Driver,
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("error sending message: %v", err)
		return
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
