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

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
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

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	durable := queueType == Durable
	autoDelete := queueType == Transient
	exclusive := queueType == Transient

	queue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	})

	if err != nil {
		return nil, amqp.Queue{}, err
	}

	if err = ch.QueueBind(queueName, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {

	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)

	if err != nil {
		return err
	}

	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)

	if err != nil {
		return err
	}

	go func() {
		for delivery := range deliveries {
			var unMarshalledDelivery T

			if err := json.Unmarshal(delivery.Body, &unMarshalledDelivery); err != nil {
				log.Fatalf("Error unmarshalling the delivery body %v", err)
			}

			acktype := handler(unMarshalledDelivery)

			if acktype == Ack {
				delivery.Ack(false)
				log.Print("ACK")
			}

			if acktype == NackRequeue {
				delivery.Nack(false, true)
				log.Print("NACK and Requeue")
			}

			if acktype == NackDiscard {
				delivery.Nack(false, false)
				log.Print("Nack and Discard")
			}
		}
	}()

	return nil
}
