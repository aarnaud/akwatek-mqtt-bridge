package models

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type ValveAction int

const (
	VALVE_ACTION_OPEN  ValveAction = 1
	VALVE_ACTION_CLOSE ValveAction = 0
)

type AkwatekCtl struct {
	MAC         net.HardwareAddr     `json:"-"`
	Value       []byte               `json:"-"`
	Sensors     map[int]*LeakoSensor `json:"-"`
	ValveAction *ValveAction         `json:"-"`
}

func NewAkwatekCtl(v1 *ReqItekV1) (*AkwatekCtl, error) {
	akwatekCtl := AkwatekCtl{
		MAC:     v1.MacAddress,
		Sensors: map[int]*LeakoSensor{},
	}
	if err := akwatekCtl.Parse(v1); err != nil {
		return nil, err
	}
	return &akwatekCtl, nil
}

func (a *AkwatekCtl) Parse(v1 *ReqItekV1) error {
	rawHex := make([]byte, 0)
	for _, digit := range []byte(v1.CtlStatus) {
		rawDecode, err := strconv.ParseInt(string(digit), 16, 8)
		if err != nil {
			return err
		}
		rawHex = append(rawHex, uint8(rawDecode))
	}

	if err := a.ParseSensors(v1); err != nil {
		return err
	}

	a.Value = rawHex
	return nil
}

func (a *AkwatekCtl) ParseSensors(v1 *ReqItekV1) error {
	rawSensors := []byte(v1.Zone01To25)
	rawSensors = append(rawSensors, []byte(v1.Zone26To50)...)
	rawSensors = append(rawSensors, []byte(v1.Zone51To75)...)
	rawSensors = append(rawSensors, []byte(v1.Zone76To100)...)

	for id, rawSensor := range rawSensors {
		raw, err := strconv.ParseInt(string(rawSensor), 16, 8)
		if err != nil {
			return err
		}
		if raw == 0x0 {
			continue
		}
		a.Sensors[id+1] = &LeakoSensor{
			ID:    id + 1,
			Value: uint8(raw),
			Ctl:   a,
		}
	}
	return nil
}

func (a *AkwatekCtl) HasPowerLine() bool {
	return a.Value[0]&0b1 == 0b1
}

func (a *AkwatekCtl) IsValveOpen() bool {
	valveOpen := a.Value[4]&0b1 == 0b1

	// Handle the 2min delay feedback for valve action
	valveActionClose := VALVE_ACTION_CLOSE
	if a.ValveAction != nil &&
		a.ValveAction == &valveActionClose &&
		valveOpen {
		return false
	}
	// Handle the 2min delay feedback for valve action if no Alarm (can't open remotely)
	valveActionOpen := VALVE_ACTION_OPEN
	if a.ValveAction != nil &&
		a.ValveAction == &valveActionOpen &&
		!valveOpen && !a.HasAlarm() {
		return true
	}

	return valveOpen
}

func (a *AkwatekCtl) HasAlarm() bool {
	return a.Value[2]&0b1 == 0b1
}

func (a *AkwatekCtl) HasBattery() bool {
	return a.Value[1]&0b1000 == 0b1000
}

func (a *AkwatekCtl) String() string {
	return fmt.Sprintf("%s power=%t battery=%t valve=%t alarm=%t", a.MAC.String(), a.HasPowerLine(), a.HasBattery(), a.IsValveOpen(), a.HasAlarm())
}

func (a *AkwatekCtl) GetIdentifier() string {
	return strings.ReplaceAll(a.MAC.String(), ":", "-")
}

func (a *AkwatekCtl) ValveCallback() func(ValveAction) {
	return func(value ValveAction) {
		a.ValveAction = &value
	}
}

func (a *AkwatekCtl) GetMQTTAvailabilityTopic() string {
	return fmt.Sprintf("%s/controller/availability", a.GetIdentifier())
}

func (a *AkwatekCtl) GetMQTTStateTopic() string {
	return fmt.Sprintf("%s/controller/state", a.GetIdentifier())
}

func (a *AkwatekCtl) MarshalJSON() ([]byte, error) {
	type Alias AkwatekCtl
	alias := (*Alias)(a)

	return json.Marshal(&struct {
		Mac       string `json:"mac"`
		ValveOpen bool   `json:"valve"`
		Battery   bool   `json:"battery"`
		PowerLine bool   `json:"powerLine"`
		Alarm     bool   `json:"alarm"`
		*Alias
	}{
		Mac:       a.MAC.String(),
		ValveOpen: a.IsValveOpen(),
		Battery:   a.HasBattery(),
		PowerLine: a.HasPowerLine(),
		Alarm:     a.HasAlarm(),
		Alias:     alias,
	})
}

type LeakoSensor struct {
	ID    int         `json:"id"`
	Value byte        `json:"-"`
	Ctl   *AkwatekCtl `json:"-"`
}

func (a *LeakoSensor) IsWaterDetected() bool {
	return a.Value&0b1000 == 0b1000
}

func (a *LeakoSensor) IsBatLow() bool {
	return a.Value&0b0100 == 0b0100
}

func (a *LeakoSensor) IsConfigured() bool {
	return a.Value&0b0001 == 0b0001
}

func (a *LeakoSensor) String() string {
	return fmt.Sprintf("ID:%d Leak=%t LowBat=%t Configured=%t", a.ID, a.IsWaterDetected(), a.IsBatLow(), a.IsConfigured())
}

func (a *LeakoSensor) GetMQTTAvailabilityTopic() string {
	return fmt.Sprintf("%s/sensors/%d/availability", a.Ctl.GetIdentifier(), a.ID)
}

func (a *LeakoSensor) GetMQTTStateTopic() string {
	return fmt.Sprintf("%s/sensors/%d/state", a.Ctl.GetIdentifier(), a.ID)
}

func (a *LeakoSensor) MarshalJSON() ([]byte, error) {
	type Alias LeakoSensor
	alias := (*Alias)(a)

	return json.Marshal(&struct {
		LowBat bool `json:"low_bat"`
		Leak   bool `json:"leak"`
		*Alias
	}{
		LowBat: a.IsBatLow(),
		Leak:   a.IsWaterDetected(),
		Alias:  alias,
	})
}
