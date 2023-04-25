package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Rabbit MQ config
var (
	uri = flag.String("uri", "amqp://guest:guest@host.docker.internal:5672/", "AMQP URI")
	// uri          = flag.String("uri", "amqp://guest:guest@host.docker.internal:5672/", "AMQP URI")
	exchange     = flag.String("exchange", "trade-exchange", "Durable, non-auto-deleted AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	lifetime     = flag.Duration("lifetime", 0*time.Second, "lifetime of process before shutdown (0s=infinite)")
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func MQStart() {
	log.Println("MQ started")
	key := fmt.Sprintf("%s-key", *pairs)
	queueName := fmt.Sprintf("%s-queue", *pairs)
	comsumerTag := fmt.Sprintf("%s-consumer", *pairs)
	c, err := NewConsumer(*uri, *exchange, *exchangeType, queueName, key, comsumerTag)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if *lifetime > 0 {
		// log.Printf("running for %s", *lifetime)
		time.Sleep(*lifetime)
	} else {
		// log.Printf("running forever")
		select {}
	}

	// log.Printf("shutting down")

	if err := c.Shutdown(); err != nil {
		log.Fatalf("error during shutdown: %s", err)
	}
}

func NewConsumer(amqpURI, exchange, exchangeType, queueName, key, ctag string) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	var err error

	// // log.Printf("dialing %q", amqpURI)
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	// // log.Printf("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	// // log.Printf("got Channel, declaring Exchange (%q)", exchange)
	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	// // log.Printf("declared Exchange, declaring Queue %q", queueName)
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	// log.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
	// 	queue.Name, queue.Messages, queue.Consumers, key)

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	// log.Printf("Queue bound to Exchange, starting Consume (consumer tag %q)", c.tag)
	deliveries, err := c.channel.Consume(
		queue.Name, // name
		c.tag,      // consumerTag,
		false,      // noAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Consume: %s", err)
	}

	go handle(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	// defer log.Printf("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

func publish(amqpURI, exchange, exchangeType, routingKey, body string, reliable bool) error {

	// This function dials, connects, declares, publishes, and tears down,
	// all in one go. In a real service, you probably want to maintain a
	// long-lived connection as state, and publish against that.

	// log.Printf("dialing %q", amqpURI)
	connection, err := amqp.Dial(amqpURI)
	if err != nil {
		return fmt.Errorf("Dial: %s", err)
	}
	defer connection.Close()

	// log.Printf("got Connection, getting Channel")
	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	// log.Printf("got Channel, declaring %q Exchange (%q)", exchangeType, exchange)
	if err := channel.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if reliable {
		// log.Printf("enabling publishing confirms.")
		if err := channel.Confirm(false); err != nil {
			return fmt.Errorf("Channel could not be put into confirm mode: %s", err)
		}

		confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

		defer confirmOne(confirms)
	}

	// // log.Printf("declared Exchange, publishing %dB body (%q)", len(body), body)
	if err = channel.Publish(
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            []byte(body),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

// One would typically keep a channel of publishings, a sequence number, and a
// set of unacknowledged sequence numbers and loop until the publishing channel
// is closed.
func confirmOne(confirms <-chan amqp.Confirmation) {
	// log.Printf("waiting for confirmation of one publishing")

	if confirmed := <-confirms; confirmed.Ack {
		// log.Printf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
	} else {
		// log.Printf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
	}
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		start := time.Now()

		type args struct {
			OrderId    string `json:"order_id"`
			OrderType  string `json:"order_type"`
			PriceType  string `json:"price_type"`
			Price      string `json:"price"`
			Quantity   string `json:"quantity"`
			Amount     string `json:"amount"`
			CreateTime int64  `json:"create_time"`
		}
		// Parsing json data
		var param args
		err := json.Unmarshal([]byte(d.Body), &param)
		if err != nil {
			logrus.Println(err)
		}

		logrus.Infof("%v", param)

		pt := PriceTypeLimit
		ts := time.Now().UnixNano()

		// formatting the order object
		param.CreateTime = ts

		if strings.ToLower(param.OrderType) == "ask" {
			item := NewAskItem(pt, param.OrderId, string2decimal(param.Price), string2decimal(param.Quantity), string2decimal(param.Amount), ts)
			btcusdt.ChNewOrder <- item

		} else {
			item := NewBidItem(pt, param.OrderId, string2decimal(param.Price), string2decimal(param.Quantity), string2decimal(param.Amount), ts)
			btcusdt.ChNewOrder <- item
		}

		go sendMessage("new_order", param)
		logrus.Printf("%v", param)

		elapsed := time.Since(start)
		logrus.Printf("time elapse: %s", elapsed)
		d.Ack(false)
	}
	// log.Printf("handle: deliveries channel closed")
	done <- nil
}

func pushToOutputqueue(completedOrder string) {
	// conn, err := amqp.Dial("amqp://guest:guest@host.docker.internal:5672/")
	conn, err := amqp.Dial("amqp://guest:guest@host.docker.internal:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	queueName := "output-queue"
	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}
	message := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(completedOrder),
	}
	// Publish the message to the queue
	err = ch.Publish("", queueName, false, false, message)
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	log.Println("Message sent to queue:", queueName)

}
