package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	marshalledValue, err := json.Marshal(val)

	if err != nil {
		log.Fatalf("Error in marshalling value %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "json",
		Body:        marshalledValue,
	})
}
