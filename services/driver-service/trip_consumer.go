package main

import (
	"context"
	"log"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type tripEventConsumer struct {
	rabbitmq *messaging.RabbitMQ
}

func newTripEventConsumer(rabbitmq *messaging.RabbitMQ) *tripEventConsumer {
	return &tripEventConsumer{rabbitmq: rabbitmq}
}

func (t *tripEventConsumer) Listen() error {
	return t.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp.Delivery) error {
		log.Printf("Received message: %s", msg.Body)
		return nil
	})
}
