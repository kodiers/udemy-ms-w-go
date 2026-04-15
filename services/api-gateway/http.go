package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

var tracer = tracing.GetTracer("api-gateway")

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview")
	defer span.End()
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if reqBody.UserId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	log.Println(reqBody)

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tripService.Close()
	tripPreview, err := tripService.Client.PreviewTrip(ctx, reqBody.ToProto())
	if err != nil {
		log.Printf("failed to preview trip: %v", err)
		http.Error(w, "Failed to preview trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{
		Data: tripPreview,
	}
	writeJson(w, http.StatusCreated, response)

}

func handleTripCreate(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripCreate")
	defer span.End()
	var reqBody startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tripService.Close()
	trip, err := tripService.Client.CreateTrip(ctx, reqBody.ToProto())
	if err != nil {
		log.Printf("failed to create trip: %v", err)
		http.Error(w, "Failed to create trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{
		Data: trip,
	}
	writeJson(w, http.StatusCreated, response)

}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx, span := tracer.Start(r.Context(), "handleStripeWebhook")
	defer span.End()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")
	if webhookKey == "" {
		http.Error(w, "Missing Stripe webhook key", http.StatusInternalServerError)
		return
	}
	event, err := webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), webhookKey,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		log.Printf("failed to construct webhook event: %v", err)
		http.Error(w, "Failed to process Stripe webhook", http.StatusInternalServerError)
		return
	}
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		err = json.Unmarshal(event.Data.Raw, &sess)
		if err != nil {
			log.Printf("failed to unmarshal checkout session: %v", err)
			http.Error(w, "Failed to process Stripe webhook", http.StatusInternalServerError)
			return
		}
		payload := messaging.PaymentStatusUpdateData{
			TripID:   sess.Metadata["trip_id"],
			UserID:   sess.Metadata["user_id"],
			DriverID: sess.Metadata["driver_id"],
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: sess.Metadata["user_id"],
			Data:    payloadBytes,
		}
		if err = rb.PublishMessage(ctx, contracts.PaymentEventSuccess, message); err != nil {
			log.Printf("failed to publish payment success event: %v", err)
			http.Error(w, "Failed to process Stripe webhook", http.StatusInternalServerError)
			return
		}
	}
}
