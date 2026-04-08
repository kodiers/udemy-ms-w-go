package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type tripEventConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func newTripEventConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripEventConsumer {
	return &tripEventConsumer{rabbitmq: rabbitmq, service: service}
}

func (t *tripEventConsumer) Listen() error {
	return t.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var tripEvent contracts.AmqpMessage
		err := json.Unmarshal(msg.Body, &tripEvent)
		if err != nil {
			return err
		}
		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			return err
		}
		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return t.handleFindAndNotifyDrivers(ctx, payload)
		}
		log.Printf("Unknown message: %s", msg.Body)
		return nil
	})
}

func (t *tripEventConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitable := t.service.FindAvailableDriversId(payload.Trip.SelectedFare.PackageSlug)

	if len(suitable) == 0 {
		if err := t.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserId,
			Data:    nil,
		}); err != nil {
			return err
		}
		return nil
	}
	randomIndex := rand.IntN(len(suitable))
	suitableDriverId := suitable[randomIndex]
	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := t.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverId,
		Data:    marshalledEvent,
	}); err != nil {
		return err
	}
	return nil
}
