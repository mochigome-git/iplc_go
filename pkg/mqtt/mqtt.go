package mqtt

import (
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func NewMQTTClient(mqttHost string, logger *log.Logger) MQTT.Client {
	mqttclient := MQTT.NewClient(MQTT.NewClientOptions().AddBroker(mqttHost))
	if token := mqttclient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("Error connecting to MQTT server: %s", token.Error())
	} else {
		logger.Printf("Connected to MQTT server %s successfully", mqttHost)
	}
	return mqttclient
}

func PublishMessage(client MQTT.Client, topic string, message string, logger *log.Logger) {
	token := client.Publish(topic, 0, false, message)
	token.Wait()
	if token.Error() != nil {
		logger.Printf("Error publishing message to topic %s: %s", topic, token.Error())
	} else {
		logger.Printf("Published message to topic %s: %s", topic, message)
	}
}
