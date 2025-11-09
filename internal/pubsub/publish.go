package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var gobedValue bytes.Buffer
	enc := gob.NewEncoder(&gobedValue)

	if err := enc.Encode(val); err != nil {
		log.Fatalf("Error in gobbingn value %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "json",
		Body:        gobedValue.Bytes(),
	})
}

func PublishGameLog(ch *amqp.Channel, message, userName string) AckType {
	routingKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, userName)

	if err := PublishGob(ch, routing.ExchangePerilTopic, routingKey, routing.GameLog{
		CurrentTime: time.Now(),
		Message:     message,
		Username:    userName,
	}); err != nil {
		return NackRequeue
	}

	return Ack
}
