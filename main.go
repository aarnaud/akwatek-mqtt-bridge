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
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	config := utils.GetConfig()
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
			go cli.WatchValve(itekv1.GetIdentifier(), ctl.ValveCallback())
		} else { // if exist, update values of controller
			if err := ctlList[itekv1.GetIdentifier()].Parse(&itekv1); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{})
				return
			}
		}

		log.Info().Msgf("%v -- %v -- %v", reqBodyItekV1.ItekV1, ctlList[itekv1.GetIdentifier()].Sensors, ctlList[itekv1.GetIdentifier()])
		c.JSON(http.StatusOK, models.ResBodyItekV1{
			ItekV1: models.ResItekV1{
				Message: "OK",
				Valve:   ctlList[itekv1.GetIdentifier()].ValveAction,
			},
		})

		cli.PublishAvailability(ctlList[itekv1.GetIdentifier()].GetMQTTAvailabilityTopic())
		cli.PublishState(ctlList[itekv1.GetIdentifier()].GetMQTTStateTopic(), ctlList[itekv1.GetIdentifier()])
		for _, sensor := range ctlList[itekv1.GetIdentifier()].Sensors {
			cli.PublishAvailability(sensor.GetMQTTAvailabilityTopic())
			cli.PublishState(sensor.GetMQTTStateTopic(), sensor)
		}
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
