package messaging

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

type MessageHandler func(ctx context.Context, msg amqp.Delivery) error

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	rmq := &RabbitMQ{conn: conn, Channel: ch}
	if err := rmq.setupExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, err
	}
	return rmq, nil
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message string) error {
	return r.Channel.PublishWithContext(ctx, "", "hello", false, false,
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(message),
			DeliveryMode: amqp.Persistent,
		})
}

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	err := r.Channel.Qos(1, 0, false)
	if err != nil {
		return err
	}
	msgs, err := r.Channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	ctx := context.Background()

	go func() {
		for msg := range msgs {
			if err := handler(ctx, msg); err != nil {
				log.Printf("error handling message: %v", err)
				if nackErr := msg.Nack(false, true); nackErr != nil {
					log.Printf("error nacking message: %v", nackErr)
				}
				continue
			}
			if ackErr := msg.Ack(true); ackErr != nil {
				log.Printf("error acking message: %v", ackErr)
			}
		}
	}()

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	_, err := r.Channel.QueueDeclare("hello", true, false, false, false, nil)
	if err != nil {
		return err
	}
	return nil
}
