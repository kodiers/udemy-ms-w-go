package events

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"

	amqp "github.com/rabbitmq/amqp091-go"
)

type driverResponseConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverResponseConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverResponseConsumer {
	return &driverResponseConsumer{rabbitmq: rabbitmq, service: service}
}

func (c *driverResponseConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverCmdTripResponseQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var message contracts.AmqpMessage
		err := json.Unmarshal(msg.Body, &message)
		if err != nil {
			return err
		}
		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			return err
		}
		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := c.handleTripAccepted(ctx, payload.TripID, payload.Driver); err != nil {
				return err
			}
		case contracts.DriverCmdTripDecline:
			log.Printf("Trip declined: %s", payload.TripID)
			if err := c.handleTripDeclined(ctx, payload.TripID, payload.RiderID); err != nil {
				return err
			}
		}
		log.Printf("Unknown message: %s", msg.Body)
		return nil
	})
}

func (c *driverResponseConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pb.Driver) error {
	trip, err := c.service.GetTripById(ctx, tripID)
	if err != nil {
		return err
	}
	if trip == nil {
		return errors.New("trip not found")
	}
	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		return err
	}
	trip, err = c.service.GetTripById(ctx, tripID)
	if err != nil {
		return err
	}
	marshalledTrip, err := json.Marshal(trip)
	if err != nil {
		return err
	}
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledTrip,
	}); err != nil {
		return err
	}
	marshalledPayload, err := json.Marshal(messaging.PaymentTripResponseData{
		TripID:   tripID,
		UserID:   trip.UserID,
		DriverID: driver.Id,
		Amount:   trip.RideFare.TotalPriceInCents,
		Currency: "USD",
	})

	if err := c.rabbitmq.PublishMessage(ctx, contracts.PaymentCmdCreateSession,
		contracts.AmqpMessage{
			OwnerID: trip.UserID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}
	return nil
}

func (c *driverResponseConsumer) handleTripDeclined(ctx context.Context, tripID string, riderId string) error {
	trip, err := c.service.GetTripById(ctx, tripID)
	if err != nil {
		return err
	}
	if trip == nil {
		return errors.New("trip not found")
	}
	newPayload := messaging.TripEventData{Trip: trip.ToProto()}
	marshalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested, contracts.AmqpMessage{
		OwnerID: riderId,
		Data:    marshalledPayload,
	}); err != nil {
		return err
	}
	return nil
}
