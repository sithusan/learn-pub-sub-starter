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

	gameState := gamelogic.NewGameState(userName)

	pauseQueueName := fmt.Sprintf("%s.%s", routing.PauseKey, userName)
	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		pauseQueueName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gameState),
	)

	armyMoveQueueName := fmt.Sprintf("army_moves.%s", userName)
	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueueName,
		"army_moves.*",
		pubsub.Transient,
		handlerMove(gameState, ch),
	)

	warQueueName := "war"
	warQueueRoutingKey := fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix)
	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		warQueueName,
		warQueueRoutingKey,
		pubsub.Durable,
		handlerWar(gameState),
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

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Println()
		moveOutCome := gs.HandleMove(am)

		fmt.Println("move outcome", moveOutCome)

		if moveOutCome == gamelogic.MoveOutcomeSamePlayer {
			return pubsub.NackDiscard
		}

		if moveOutCome == gamelogic.MoveOutComeSafe {
			return pubsub.Ack
		}

		if moveOutCome == gamelogic.MoveOutcomeMakeWar {
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.Player.Username)

			if err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routingKey,
				gamelogic.RecognitionOfWar{
					Attacker: am.Player,
					Defender: gs.GetPlayerSnap(),
				},
			); err != nil {
				log.Printf("Could not publish move outcome make war %v", err)
			}

			return pubsub.NackRequeue
		}

		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState) func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		outcome, _, _ := gs.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			log.Print("Outcome not recognized")
			return pubsub.NackDiscard
		}
	}
}
