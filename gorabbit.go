// Base of idea to https://github.com/pandeptwidyaop/gorabbit. Thanks dude!
// Added method ConnectTLS()

package gorabbit

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/streadway/amqp"
)

var (
	Context *RabbitMQ

	ErrConRefused   = errors.New("connection refused by server")
	ErrNotConnected = errors.New("not connected to server")
)

type RabbitMQ struct {
	connection       *amqp.Connection
	channel          *amqp.Channel
	queue            amqp.Queue
	err              chan error
	connectionString string
	exchange         string
	bindings         []string
	name             string
}

// New instance
func New(connectionString string, name string, exchange string) (*RabbitMQ, error) {
	mq := RabbitMQ{
		err:              make(chan error),
		connectionString: connectionString,
		exchange:         exchange,
		name:             name,
	}

	Context = &mq

	return &mq, nil
}

func (mq *RabbitMQ) Connect() error {
	var err error
	log.Println("[gorabbit] connecting to server")
	mq.connection, err = amqp.Dial(mq.connectionString)

	if err != nil {
		return err
	}

	mq.channel, err = mq.connection.Channel()

	if err != nil {
		return err
	}

	go func() {
		<-mq.channel.NotifyClose(make(chan *amqp.Error))
		mq.err <- ErrConRefused
	}()

	return nil
}

func (mq *RabbitMQ) Bind(bindings []string) error {
	var err error
	if mq.channel == nil {
		return ErrNotConnected
	}

	mq.bindings = bindings

	mq.queue, err = mq.channel.QueueDeclare(
		mq.name,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	for _, bind := range mq.bindings {
		log.Printf("[gorabbit] binding %s to channel \n", bind)
		err = mq.channel.QueueBind(
			mq.queue.Name,
			bind,
			mq.exchange,
			false,
			nil,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (mq *RabbitMQ) Reconnect() error {
	log.Println("[gorabbit] reconnecting to server")
	var err error

	err = mq.Connect()
	if err != nil {
		return err
	}

	err = mq.Bind(mq.bindings)
	if err != nil {
		return err
	}

	return nil
}

func (mq *RabbitMQ) Consume() (map[string]<-chan amqp.Delivery, error) {
	m := make(map[string]<-chan amqp.Delivery)
	for _, q := range mq.bindings {
		deliveries, err := mq.channel.Consume(
			mq.queue.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return nil, err
		}
		m[q] = deliveries
	}
	return m, nil
}

func (mq *RabbitMQ) HandleConsumedDeliveries(q string, delivery <-chan amqp.Delivery, fn func(RabbitMQ, string, <-chan amqp.Delivery)) {
	for {
		go fn(*mq, q, delivery)
		if err := <-mq.err; err != nil {
			mq.Reconnect()
			deliveries, err := mq.Consume()
			if err != nil {
				panic(err)
			}

			delivery = deliveries[q]
		}
	}
}

func (mq *RabbitMQ) Publish(event string, contentType string, message []byte) error {
	select {
	case err := <-mq.err:
		if err != nil {
			mq.Reconnect()
		}
	default:
	}

	m := amqp.Publishing{
		ContentType: contentType,
		Body:        message,
	}

	return mq.channel.Publish(
		mq.exchange,
		event,
		false,
		false,
		m,
	)
}

// Method of TLS.
func (mq *RabbitMQ) ConnectTLS() error {
	var err error
	cfg := new(tls.Config)

	// Checks to cert path, that must be
	_, err = os.Open("./cacert.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File cacert.pem not found: %v", err)
	}

	// Certificate must be
	cfg.RootCAs = x509.NewCertPool()
	if ca, err := ioutil.ReadFile("./cacert.pem"); err == nil {
		cfg.RootCAs.AppendCertsFromPEM(ca)
	}

	// Checks to certs paths, that must be
	_, err = os.Open("./cert.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File cert.pem not found: %v", err)
	}
	_, err = os.Open("./key.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File key.pem not found: %v", err)
	}

	// Loads client cert and key.
	if cert, err := tls.LoadX509KeyPair("./cert.pem", "./key.pem"); err == nil {
		cfg.Certificates = append(cfg.Certificates, cert)
	}
	log.Println("[tlsgorabbit] connecting to server")

	// One should use Common Name (CN) PC, it a server name from certificate
	conn, err := amqp.DialTLS("amqps://server-name-from-certificate", cfg)
	if err != nil {
		log.Fatalf("Error conn of server: %v", err)
	}

	mq.channel, err = conn.Channel()
	if err != nil {
		return err
	}

	go func() {
		<-mq.channel.NotifyClose(make(chan *amqp.Error))
		mq.err <- ErrConRefused
	}()

	return nil
}
