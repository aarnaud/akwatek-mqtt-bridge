package utils

import (
	"github.com/spf13/viper"
)

type Config struct {
	TLSPort            int
	MQTT               *ConfigMQTT
	HassDiscoveryTopic string
}

type ConfigMQTT struct {
	BrokerHost string
	BrokerPort int
	ClientID   string
	BaseTopic  string
	Username   string
	Password   string
}

func GetConfig() *Config {
	// the env registry will look for env variables that start with "OMB_".
	viper.SetEnvPrefix("AMB")
	// Enable VIPER to read Environment Variables
	viper.AutomaticEnv()                       // To get the value from the config file using key// viper package read .env
	viper.SetConfigName("akwatek-mqtt-bridge") // name of config file (without extension)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	viper.SetDefault("TLS_PORT", 8443)
	viper.SetDefault("MQTT_BROKER_PORT", 1883)
	viper.SetDefault("MQTT_CLIENT_ID", "akwatek")
	viper.SetDefault("MQTT_BASE_TOPIC", "akwatek")
	viper.SetDefault("HASS_DISCOVERY_TOPIC", "homeassistant")

	config := Config{
		TLSPort: viper.GetInt("TLS_PORT"),
		MQTT: &ConfigMQTT{
			BrokerHost: viper.GetString("MQTT_BROKER_HOST"),
			BrokerPort: viper.GetInt("MQTT_BROKER_PORT"),
			ClientID:   viper.GetString("MQTT_CLIENT_ID"),
			BaseTopic:  viper.GetString("MQTT_BASE_TOPIC"),
			Username:   viper.GetString("MQTT_USERNAME"),
			Password:   viper.GetString("MQTT_PASSWORD"),
		},
		HassDiscoveryTopic: viper.GetString("HASS_DISCOVERY_TOPIC"),
	}
	return &config
}
