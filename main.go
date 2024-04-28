package main

import (
	"akwatek-mqtt-bridge/models"
	mqtt_client "akwatek-mqtt-bridge/mqtt-client"
	"akwatek-mqtt-bridge/utils"
	"crypto/tls"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

func main() {

	config := utils.GetConfig()
	zerolog.SetGlobalLevel(config.LogLevel)
	cli := mqtt_client.NewMQTT(config)
	router := gin.New()

	ctlList := make(map[string]*models.AkwatekCtl)
	router.POST("/collect2.php", func(c *gin.Context) {
		var reqBodyItekV1 models.ReqBodyItekV1
		if err := c.Bind(&reqBodyItekV1); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{})
			return
		}
		itekv1 := reqBodyItekV1.ItekV1
		// create new akwatek controller object if not present
		if _, ok := ctlList[itekv1.GetIdentifier()]; !ok {
			ctl, err := models.NewAkwatekCtl(&itekv1)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{})
				return
			}
			ctlList[itekv1.GetIdentifier()] = ctl
			go cli.WatchValve(ctl.GetMQTTSValveCommandTopic(config.MQTT.BaseTopic), ctl.ValveCallback())
		} else { // if exist, update values of controller
			if err := ctlList[itekv1.GetIdentifier()].Parse(&itekv1); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{})
				return
			}
		}

		ctl := ctlList[itekv1.GetIdentifier()]

		log.Debug().Msgf("%v", reqBodyItekV1.ItekV1)
		log.Info().Msgf("%s -- %v", ctl, ctl.Sensors)
		c.JSON(http.StatusOK, models.ResBodyItekV1{
			ItekV1: models.ResItekV1{
				Message: "OK",
				Valve:   ctlList[itekv1.GetIdentifier()].GetValveAction(),
			},
		})

		// Async mqtt publish
		go func() {
			if ctl.LastHassConfigPublished.Add(time.Hour).Before(time.Now()) {
				PublishHassConfig(config, cli, ctl)
				// delay before send state
				time.Sleep(time.Second * 5)
			}

			cli.PublishAvailability(ctl.GetMQTTAvailabilityTopic(config.MQTT.BaseTopic))
			cli.PublishState(ctl.GetMQTTStateTopic(config.MQTT.BaseTopic), ctl)
			for _, sensor := range ctl.Sensors {
				if !sensor.IsConfigured() {
					continue
				}
				cli.PublishAvailability(sensor.GetMQTTAvailabilityTopic(config.MQTT.BaseTopic))
				cli.PublishState(sensor.GetMQTTStateTopic(config.MQTT.BaseTopic), sensor)
			}
			// don't repeat valve action on the next call
			ctlList[itekv1.GetIdentifier()].ResetValveAction()
		}()
	})

	// get our ca and server certificate
	serverTLSConf, _, err := utils.CertSetup()
	if err != nil {
		panic(err)
	}

	tlsServer, err := tls.Listen("tcp", fmt.Sprintf(":%d", config.TLSPort), serverTLSConf)
	if err != nil {
		panic(err)
	}

	router.RunListener(tlsServer)
}

func PublishHassConfig(config *utils.Config, cli *mqtt_client.Client, ctl *models.AkwatekCtl) {
	log.Info().Msgf("Publishing homeassistant mqtt config")
	cli.PublishState(
		ctl.GetMQTTValveHassConfigTopic(config.HassDiscoveryTopic),
		ctl.GetMQTTValveHassConfig(config.MQTT.BaseTopic))
	// delay for home-assistant on the first time creation
	time.Sleep(time.Second * 5)
	cli.PublishState(
		ctl.GetMQTTAlarmHassConfigTopic(config.HassDiscoveryTopic),
		ctl.GetMQTTAlarmHassConfig(config.MQTT.BaseTopic))
	cli.PublishState(
		ctl.GetMQTTPowerHassConfigTopic(config.HassDiscoveryTopic),
		ctl.GetMQTTPowerHassConfig(config.MQTT.BaseTopic))
	cli.PublishState(
		ctl.GetMQTTBatteryHassConfigTopic(config.HassDiscoveryTopic),
		ctl.GetMQTTBatteryHassConfig(config.MQTT.BaseTopic))

	for _, sensor := range ctl.Sensors {
		if !sensor.IsConfigured() {
			continue
		}
		cli.PublishState(
			sensor.GetMQTTBatHassConfigTopic(config.HassDiscoveryTopic),
			sensor.GetMQTTBatHassConfig(config.MQTT.BaseTopic))
		cli.PublishState(
			sensor.GetMQTTLeakHassConfigTopic(config.HassDiscoveryTopic),
			sensor.GetMQTTLeakHassConfig(config.MQTT.BaseTopic))
		cli.PublishState(
			sensor.GetMQTTSignalHassConfigTopic(config.HassDiscoveryTopic),
			sensor.GetMQTTSignalHassConfig(config.MQTT.BaseTopic))
	}
	ctl.LastHassConfigPublished = time.Now()
}
