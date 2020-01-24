package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/firmata"
)

const (
	awsIotHostName           = "ssl://%s:8883"
	shadowUpdateEndpointBase = "$aws/things/%s/shadow/update"
	sensorOnMessage          = `{"state":{"reported":{"sensor":"on"}}}`
	sensorOffMessage         = `{"state":{"reported":{"sensor":"off"}}}`
)

var (
	hostName             string
	clientID             string
	clientCertificate    string
	caCertificate        string
	privateKey           string
	thingName            string
	shadowUpdateEndpoint string
)

func parseArgs() {
	flag.StringVar(&hostName, "hostName", "", "host name flag")
	flag.StringVar(&clientID, "client-id", "", "client-id flag")
	flag.StringVar(&clientCertificate, "client-certificate", "", "client-certificate flag")
	flag.StringVar(&caCertificate, "ca-certificate", "", "ca-certificate flag")
	flag.StringVar(&privateKey, "private-key", "", "private-key flag")
	flag.StringVar(&thingName, "thing-name", "", "thing-name flag")
	flag.Parse()
	shadowUpdateEndpoint = fmt.Sprintf(shadowUpdateEndpointBase, thingName)
}

func newTLSConfig() (*tls.Config, error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(clientCertificate)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	cert, err := tls.LoadX509KeyPair(caCertificate, privateKey)
	if err != nil {
		return nil, err
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
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
	firmataAdaptor := firmata.NewAdaptor("/dev/ttyACM0")

	sensor := gpio.NewPIRMotionDriver(firmataAdaptor, "5")

	work := func() {
		sensor.On(gpio.MotionDetected, func(data interface{}) {
			fmt.Println(gpio.MotionDetected)
			token := c.Publish(fmt.Sprintf(shadowUpdateEndpoint), 0, false, sensorOnMessage)
			if token.Wait() && token.Error() != nil {
				fmt.Printf("error: %+v", token.Error())
			} else {
				fmt.Print("Message Publish Success\n")
			}
		})
		sensor.On(gpio.MotionStopped, func(data interface{}) {
			fmt.Println(gpio.MotionStopped)
			token := c.Publish(shadowUpdateEndpoint, 0, false, sensorOffMessage)
			if token.Wait() && token.Error() != nil {
				fmt.Printf("error: %+v", token.Error())
			} else {
				fmt.Print("Message Publish Success\n")
			}
		})
	}

	return gobot.NewRobot("motionBot",
		[]gobot.Connection{firmataAdaptor},
		[]gobot.Device{sensor},
		work,
	)
}

func main() {
	parseArgs()
	c, err := mqttClient()
	if err != nil {
		panic(err)
	}
	defer c.Disconnect(250)

	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Print("AWS IoT Connect Success\n")

	robot := newRobot(c)
	robot.Start()
}