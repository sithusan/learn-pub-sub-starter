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

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("Error in opening channel %v", err)
	}

	gamelogic.PrintServerHelp()

	for {
		firstWord := gamelogic.GetInput()[0]

		switch firstWord {
		case routing.PauseKey:
			log.Print("Publishing pause game state")
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				})

			if err != nil {
				log.Printf("Could not publish time: %v", err)
			}
		case routing.ResumeKey:
			log.Print("Publishing resume game state")
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				})

			if err != nil {
				log.Printf("Could not publish time: %v", err)
			}
		case "quit":
			log.Print("Good Bye!")
			return

		default:
			log.Print("Unknow command")
		}

	}

	// wait for keyboard interpret
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("Connection to RabbitMQ was closed")
}
