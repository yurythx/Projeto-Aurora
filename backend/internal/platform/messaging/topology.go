package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ExchangeEvents é o único exchange topic da plataforma Projeto Aurora.
const ExchangeEvents = "aurora.events"

type QueueSpec struct {
	Name        string
	DLQName     string
	RoutingKeys []string
}

const RetryHeader = "x-aurora-attempt"

var (
	QueueNotificationWebsocket = QueueSpec{
		Name:    "aurora.notification.websocket",
		DLQName: "aurora.notification.dlq",
		RoutingKeys: []string{
			"notification.created",
			"integration.status.changed",
			"example.item.created",
		},
	}

	QueueExampleWorker = QueueSpec{
		Name:        "aurora.example.worker",
		DLQName:     "aurora.example.dlq",
		RoutingKeys: []string{"example.item.created"},
	}
)

func AllQueues() []QueueSpec {
	return []QueueSpec{QueueNotificationWebsocket, QueueExampleWorker}
}

func DeclareTopology(ch *amqp.Channel, specs []QueueSpec) error {
	if err := ch.ExchangeDeclare(
		ExchangeEvents,
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	); err != nil {
		return fmt.Errorf("messaging: declare exchange %s: %w", ExchangeEvents, err)
	}

	for _, spec := range specs {
		if err := declareQueue(ch, spec); err != nil {
			return err
		}
	}
	return nil
}

func declareQueue(ch *amqp.Channel, spec QueueSpec) error {
	if _, err := ch.QueueDeclare(spec.DLQName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("messaging: declare DLQ %s: %w", spec.DLQName, err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": spec.DLQName,
	}
	if _, err := ch.QueueDeclare(spec.Name, true, false, false, false, mainArgs); err != nil {
		return fmt.Errorf("messaging: declare queue %s: %w", spec.Name, err)
	}

	for _, key := range spec.RoutingKeys {
		if err := ch.QueueBind(spec.Name, key, ExchangeEvents, false, nil); err != nil {
			return fmt.Errorf("messaging: bind queue %s to key %s: %w", spec.Name, key, err)
		}
	}
	return nil
}
