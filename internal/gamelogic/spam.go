package gamelogic

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (gs *GameState) CommandSpam(ch *amqp.Channel, words []string) error {
	if len(words) < 2 {
		return errors.New("usage: spam <spamNumber>")
	}

	spamNumber, err := strconv.Atoi(words[1])

	if err != nil {
		return fmt.Errorf("error: %s is not valid spam number", words[1])
	}

	key := fmt.Sprintf("%s.%s", routing.GameLogSlug, gs.GetUsername())

	for i := 0; i < spamNumber; i++ {
		maliciousLog := GetMaliciousLog()
		pubsub.PublishGob(ch, routing.ExchangePerilTopic, key, routing.GameLog{
			CurrentTime: time.Now(),
			Message:     maliciousLog,
			Username:    gs.GetUsername(),
		})
	}

	return nil
}
