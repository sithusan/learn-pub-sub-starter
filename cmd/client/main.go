package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connectionString := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("Error in connecting RabbitMQ %v", err)
	}
	defer conn.Close()

	fmt.Println("Connection to RabbitMQ was success")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error in getting user name %v", err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, userName)

	ch, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
	)

	if err != nil {
		log.Fatalf("Error in declaring and bind %v", err)
	}

	fmt.Println(ch, queue, queueName)

	// wait for keyboard interpret
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("Connection to RabbitMQ was closed")
}
