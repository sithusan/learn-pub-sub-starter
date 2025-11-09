package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"

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

	unmarshaller := func(body []byte) (T, error) {

		var unMarshalledDelivery T

		if err := json.Unmarshal(body, &unMarshalledDelivery); err != nil {
			return unMarshalledDelivery, err
		}

		return unMarshalledDelivery, nil
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {

	unmarshaller := func(body []byte) (T, error) {

		dec := gob.NewDecoder(bytes.NewReader(body))

		var decodedGob T

		if err := dec.Decode(&decodedGob); err != nil {
			return decodedGob, err
		}

		return decodedGob, nil
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
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

			unmarshalledVal, err := unmarshaller(delivery.Body)

			if err != nil {
				log.Fatalf("Error decoding the delivery body %v", err)
			}

			acktype := handler(unmarshalledVal)

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
