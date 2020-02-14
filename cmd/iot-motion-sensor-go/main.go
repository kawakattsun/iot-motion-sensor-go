package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
)

const (
	location           = "Asia/Tokyo"
	awsIotHostName     = "ssl://%s:8883"
	messageTemplate    = `{"sensor":"%s","detected_at":"%s"}`
	pirMotionDriverPin = "12"
)

var (
	hostName          string
	clientID          string
	clientCertificate string
	caCertificate     string
	privateKey        string
	endpoint          string
)

func parseArgs() {
	flag.StringVar(&hostName, "host-name", "", "host name flag")
	flag.StringVar(&clientID, "client-id", "", "client-id flag")
	flag.StringVar(&clientCertificate, "client-certificate", "", "client-certificate flag")
	flag.StringVar(&caCertificate, "ca-certificate", "", "ca-certificate flag")
	flag.StringVar(&privateKey, "private-key", "", "private-key flag")
	flag.StringVar(&endpoint, "endpoint", "", "endpoint flag")
	flag.Parse()
}

func newTLSConfig() (*tls.Config, error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(caCertificate)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	cert, err := tls.LoadX509KeyPair(clientCertificate, privateKey)
	if err != nil {
		log.Printf("error: cert file mismatch. clientCertificate: %s, privateKey: %s\n", clientCertificate, privateKey)
		return nil, err
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Print("error: parse certificate.\n")
		return nil, err
	}

	return &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}, nil
}

var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func mqttClient() (MQTT.Client, error) {
	tlsconfig, err := newTLSConfig()
	if err != nil {
		log.Print("error: newTLSConfig error.\n")
		return nil, err
	}

	opts := MQTT.NewClientOptions().
		AddBroker(fmt.Sprintf(awsIotHostName, hostName)).
		SetClientID(clientID).
		SetTLSConfig(tlsconfig).
		SetDefaultPublishHandler(f)

	return MQTT.NewClient(opts), nil
}

func newRobot(c MQTT.Client) *gobot.Robot {
	r := raspi.NewAdaptor()
	sensor := gpio.NewPIRMotionDriver(r, pirMotionDriverPin)

	work := func() {
		sensor.On(gpio.MotionDetected, func(data interface{}) {
			fmt.Println(gpio.MotionDetected)
			token := c.Publish(endpoint, 0, false, fmt.Sprintf(messageTemplate, "on", time.Now().Format(time.RFC3339)))
			if token.Wait() && token.Error() != nil {
				fmt.Printf("error: %+v", token.Error())
			} else {
				fmt.Print("Message Publish Success\n")
			}
		})
	}

	return gobot.NewRobot("motionBot",
		[]gobot.Connection{r},
		[]gobot.Device{sensor},
		work,
	)
}

func init() {
	loc, err := time.LoadLocation(location)
	if err != nil {
		loc = time.FixedZone(location, 9*60*60)
	}
	time.Local = loc
}

func main() {
	parseArgs()
	c, err := mqttClient()
	if err != nil {
		panic(err)
	}
	defer c.Disconnect(250)

	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("error: AWS IoT not connected.: %+v\n", c)
		panic(token.Error())
	}
	fmt.Print("AWS IoT Connect Success\n")

	robot := newRobot(c)

	go func() {
		ticker := time.NewTicker(time.Second * 15)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				msg := fmt.Sprintf(messageTemplate, "off", time.Now().Format(time.RFC3339))
				token := c.Publish(endpoint, 0, false, msg)
				if token.Wait() && token.Error() != nil {
					fmt.Printf("error: %+v", token.Error())
				} else {
					fmt.Print("Message Publish Success. timeTicker.\n")
				}
			}
		}
	}()

	robot.Start()
}
