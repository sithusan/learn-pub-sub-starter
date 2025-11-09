package main

import (
	"fmt"
	"log"

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

	key := fmt.Sprintf("%s.*", routing.GameLogSlug)

	if err := pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		key,
		pubsub.Durable,
		handlerLog(),
	); err != nil {
		log.Fatalf("could not starting consuming logs: %v", err)

	}

	gamelogic.PrintServerHelp()

	for {

		inputs := gamelogic.GetInput()

		if len(inputs) == 0 {
			continue
		}

		firstWord := inputs[0]

		switch firstWord {
		case routing.PauseKey:
			log.Print("Publishing pause game state")

			if err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				}); err != nil {

				log.Printf("Could not publish time: %v", err)
			}

		case routing.ResumeKey:
			log.Print("Publishing resume game state")
			if err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				}); err != nil {
				log.Printf("Could not publish time: %v", err)
			}
		case "quit":
			log.Print("Good Bye!")
			return

		default:
			log.Print("Unknow command")
		}
	}
}
