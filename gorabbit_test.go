package gorabbit

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/streadway/amqp"
)

func TestRabbitMQ(t *testing.T) {
	var tests = []struct {
		channel string
	}{
		{"channel_SAP_A"},
		{"channel_SAP_B"},
		{"channel_SAP_C"},
		{"channel_1"},
		{"channel_2"},
	}

	var prevchannel string
	for _, test := range tests {
		if test.channel != prevchannel {
			fmt.Printf("\n%s\n", test.channel)
			prevchannel = test.channel
		}
	}
}

func TestCertTLS(t *testing.T) {
	var err error

	// Checks to cert path, that must be
	_, err = os.Open("cacert.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File cacert.pem not found: %v", err)
	}

	// Checks to cert path, that must be
	_, err = os.Open("cert.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File cert.pem not found: %v", err)
	}

	// Checks to cert path, that must be
	_, err = os.Open("key.pem")
	fmt.Println(os.IsNotExist(err))
	if err != nil {
		log.Fatalf("File key.pem not found: %v", err)
	}

}

var rsaCertPEM, rsaKeyPEM string
var keyPairTests = []struct {
	algo string
	cert string
	key  string
}{
	{"RSA", rsaCertPEM, rsaKeyPEM},
}

func TestX509KeyPair(t *testing.T) {
	var err error
	fc, err := os.Open("cert.pem")
	if err != nil {
		log.Fatalf("File cert.pem not found: %v", err)
	}
	defer fc.Close()
	rsaCP, err := ioutil.ReadAll(fc)
	rsaCertPEM := string(rsaCP)
	fmt.Printf("rsaCertPEM: %s\n", rsaCertPEM)
	if err != nil {
		log.Fatalf("Error read cert.pem: %v", err)
	}

	fk, err := os.Open("key.pem")
	if err != nil {
		log.Fatalf("File key.pem not found: %v", err)
	}
	defer fk.Close()
	rsaKP, err := ioutil.ReadAll(fk)
	rsaKeyPEM = string(rsaKP)
	fmt.Printf("rsaKeyPEM: %s\n", rsaKeyPEM)
	if err != nil {
		log.Fatalf("Error read key.pem: %v", err)
	}

	_, err = tls.LoadX509KeyPair(
		"key.pem",
		"cert.pem",
	)
	if err != nil {
		log.Println("Error LoadX509Key: \n", err)
	}

	t.Parallel()
	var pem []byte
	for _, test := range keyPairTests {
		pem = []byte(test.key + test.cert)
		if _, err := tls.X509KeyPair(pem, pem); err == nil {
			t.Errorf("Failed to load %s cert followed by %s key: %s", test.algo, test.algo, err)
		}
		pem = []byte(test.key + test.cert)
		if _, err := tls.X509KeyPair(pem, pem); err == nil {
			t.Errorf("Failed to load %s key followed by %s cert: %s", test.algo, test.algo, err)
		}
	}
}

// Func for send data via type of structure. Для отправки данных через тип структуры.
func BenchmarkChatClient(b *testing.B) {
	ErrConRefused = errors.New("connection refused by server")
	var tests = []struct {
		channel string
	}{
		{"channel_SAP_A"},
		{"channel_SAP_B"},
		{"channel_SAP_C"},
		{"channel_1"},
		{"channel_2"},
	}

	var prevchannel string
	for _, test := range tests {
		if test.channel != prevchannel {
			fmt.Printf("\n%s\n", test.channel)
			prevchannel = test.channel
		}

		ErrConRefused = errors.New("connection refused by server")
		errs := make(chan error)
		var err error
		b.ReportAllocs()
		for i := 0; i < 3; i++ {
			// Allows not reg a sertificaties, it's for demo mode. Разрешение не регать сертификаты, для демо.
			cfg := &tls.Config{
				InsecureSkipVerify: true,
			}

			_, err = amqp.DialTLS("amqps://localhost", cfg)
			if err == nil {
				log.Fatalf("Error conn server: %v", err)
			}
			log.Println("Connect yes: ", prevchannel)

			go func() {
				<-make(chan *amqp.Error)
				errs <- ErrConRefused
			}()
		}
	}
}
