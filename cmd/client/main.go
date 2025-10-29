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

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("Error in opening channel %v", err)
	}

	fmt.Println("Connection to RabbitMQ was success")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error in getting user name %v", err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, userName)

	_, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
	)

	if err != nil {
		log.Fatalf("Error in declaring and bind %v", err)
	}

	log.Printf("Queue %v declared and bounded \n", queue.Name)

	armyMoveQueueName := fmt.Sprintf("army_moves.%s", userName)

	_, armyMoveQueue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueueName,
		"army_moves.*",
		pubsub.Transient,
	)
	if err != nil {
		log.Fatalf("Error in declaring and bind %v", err)
	}

	log.Printf("Queue %v declared and bounded \n", armyMoveQueue.Name)

	gameState := gamelogic.NewGameState(userName)

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gameState),
	)

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueueName,
		"army_moves.*",
		pubsub.Transient,
		handlerMove(gameState),
	)

	for {
		words := gamelogic.GetInput()

		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			err = gameState.CommandSpawn(words)
			if err != nil {
				log.Println(err)
				continue
			}
		case "move":
			armyMove, err := gameState.CommandMove(words)
			if err != nil {
				log.Println(err)
			}

			log.Print("Publishing army move")

			armyMoveRoutingKey := fmt.Sprintf("army_moves.%s", userName)

			if err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				armyMoveRoutingKey,
				armyMove); err != nil {
				log.Printf("Could not publish move: %v", err)
			}

			log.Println("Army moved")
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Println("Unknown Command")
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)

		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Println()
		moveOutCome := gs.HandleMove(am)

		if moveOutCome == gamelogic.MoveOutComeSafe || moveOutCome == gamelogic.MoveOutcomeMakeWar {
			return pubsub.Ack
		}

		if moveOutCome == gamelogic.MoveOutcomeSamePlayer {
			return pubsub.NackDiscard
		}

		return pubsub.NackDiscard
	}
}
