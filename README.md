# iot-motion-sensor-go
This is an application to send from the motion sensor to AWS IoT with RaspberryPi.

## Description
Language  
GO  
  
Package used:  
- gobot.io/x/gobot
- github.com/eclipse/paho.mqtt.golang

## Usage

```
./iot-motion-sensor-go \
		-host-name xxxxx.iot.ap-northeast-1.amazonaws.com \
		-client-id clientID \
		-client-certificate xxxxx-certificate.pem.crt \
		-ca-certificate root-CA.crt \
		-private-key xxxxx-private.pem.key \
		-thing-name thigName
```
