package models

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type ValveAction string

const (
	VALVE_ACTION_OPEN   ValveAction = "1"
	VALVE_ACTION_CLOSE  ValveAction = "0"
	MANUFACTURER        string      = "AKWA Technologies"
	MANUFACTURER_PREFIX string      = "akwatek"
)

type AkwatekCtl struct {
	MAC                     net.HardwareAddr     `json:"-"`
	Value                   []byte               `json:"-"`
	Sensors                 map[int]*LeakoSensor `json:"-"`
	valveAction             *ValveAction         `json:"-"`
	LastHassConfigPublished time.Time            `json:"-"`
}

func NewAkwatekCtl(v1 *ReqItekV1) (*AkwatekCtl, error) {
	akwatekCtl := AkwatekCtl{
		MAC:                     v1.MacAddress,
		Sensors:                 map[int]*LeakoSensor{},
		LastHassConfigPublished: time.UnixMicro(0),
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
	return a.Value[4]&0b1 == 0b1
}

func (a *AkwatekCtl) ValveState() string {
	valveOpen := a.IsValveOpen()

	// Handle the 2min delay feedback for valve action
	if a.valveAction != nil &&
		*a.valveAction == VALVE_ACTION_CLOSE &&
		valveOpen {
		return "closing"
	}
	// Handle the 2min delay feedback for valve action if no Alarm (can't open remotely)
	if a.valveAction != nil &&
		*a.valveAction == VALVE_ACTION_OPEN &&
		!valveOpen && !a.HasAlarm() {
		return "opening"
	}

	if valveOpen {
		return "open"
	}
	return "closed"
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
		a.valveAction = &value
	}
}

func (a *AkwatekCtl) GetValveAction() *ValveAction {
	return a.valveAction
}

func (a *AkwatekCtl) ResetValveAction() {
	a.valveAction = nil
}

func (a *AkwatekCtl) GetMQTTAvailabilityTopic(baseTopic string) string {
	return fmt.Sprintf("%s/%s/controller/availability", baseTopic, a.GetIdentifier())
}

func (a *AkwatekCtl) GetMQTTStateTopic(baseTopic string) string {
	return fmt.Sprintf("%s/%s/controller/state", baseTopic, a.GetIdentifier())
}

func (a *AkwatekCtl) GetMQTTHassNodeId() string {
	return fmt.Sprintf("%s-%s", MANUFACTURER_PREFIX, a.GetIdentifier())
}

func (a *AkwatekCtl) GetMQTTValveHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/valve/%s/valve/config", hassPrefix, a.GetMQTTHassNodeId())
}

func (a *AkwatekCtl) GetMQTTSValveCommandTopic(baseTopic string) string {
	return fmt.Sprintf("%s/%s/controller/valve/set", baseTopic, a.GetIdentifier())
}

func (a *AkwatekCtl) GetMQTTValveHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              "valve",
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "water",
		CommandTopic:      a.GetMQTTSValveCommandTopic(baseTopic),
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_valve", a.GetMQTTHassNodeId()),
		ValueTemplate:     "{{ value_json.valve_state }}",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.GetMQTTHassNodeId()},
		},
	}
}

func (a *AkwatekCtl) GetMQTTPowerHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/power/config", hassPrefix, a.GetMQTTHassNodeId())
}

func (a *AkwatekCtl) GetMQTTPowerHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              "power",
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "power",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_power", a.GetMQTTHassNodeId()),
		ValueTemplate:     "{{ value_json.powerLine | abs }}",
		PayloadOff:        "0",
		PayloadOn:         "1",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.GetMQTTHassNodeId()},
		},
	}
}

func (a *AkwatekCtl) GetMQTTBatteryHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/battery/config", hassPrefix, a.GetMQTTHassNodeId())
}

func (a *AkwatekCtl) GetMQTTBatteryHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              "battery",
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "problem",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_battery", a.GetMQTTHassNodeId()),
		ValueTemplate:     "{{ value_json.battery | abs }}",
		PayloadOff:        "1",
		PayloadOn:         "0",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.GetMQTTHassNodeId()},
		},
	}
}

func (a *AkwatekCtl) GetMQTTAlarmHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/alarm/config", hassPrefix, a.GetMQTTHassNodeId())
}

func (a *AkwatekCtl) GetMQTTAlarmHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              "alarm",
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "problem",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_alarm", a.GetMQTTHassNodeId()),
		ValueTemplate:     "{{ value_json.alarm | abs }}",
		PayloadOff:        "0",
		PayloadOn:         "1",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.GetMQTTHassNodeId()},
		},
	}
}

func (a *AkwatekCtl) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Mac        string `json:"mac"`
		ValveOpen  bool   `json:"valve"`
		ValveState string `json:"valve_state"`
		Battery    bool   `json:"battery"`
		PowerLine  bool   `json:"powerLine"`
		Alarm      bool   `json:"alarm"`
	}{
		Mac:        a.MAC.String(),
		ValveOpen:  a.IsValveOpen(),
		ValveState: a.ValveState(),
		Battery:    a.HasBattery(),
		PowerLine:  a.HasPowerLine(),
		Alarm:      a.HasAlarm(),
	})
}

type LeakoSensor struct {
	ID    int
	Value byte
	Ctl   *AkwatekCtl
}

func (a *LeakoSensor) IsWaterDetected() bool {
	return a.Value&0b1000 == 0b1000
}

func (a *LeakoSensor) IsBatLow() bool {
	return a.Value&0b0100 == 0b0100
}

func (a *LeakoSensor) IsLostSignal() bool {
	return a.Value&0b010 == 0b0010
}

func (a *LeakoSensor) IsConfigured() bool {
	return a.Value&0b0001 == 0b0001
}

func (a *LeakoSensor) String() string {
	value := strings.Builder{}
	if a.IsWaterDetected() {
		value.Write([]byte("leak"))
	}
	if a.IsBatLow() {
		if value.Len() > 0 {
			value.Write([]byte("+"))
		}
		value.Write([]byte("LowBat"))
	}
	if !a.IsConfigured() {
		if value.Len() > 0 {
			value.Write([]byte("+"))
		}
		value.Write([]byte("NotConf"))
	}
	if a.IsLostSignal() {
		if value.Len() > 0 {
			value.Write([]byte("+"))
		}
		value.Write([]byte("LostSignal"))
	}
	if value.Len() == 0 {
		value.Write([]byte("ok"))
	}
	return value.String()
}

func (a *LeakoSensor) GetIdentifier() string {
	return fmt.Sprintf("%s-%s_%d", MANUFACTURER_PREFIX, a.Ctl.GetIdentifier(), a.ID)
}

func (a *LeakoSensor) GetMQTTAvailabilityTopic(baseTopic string) string {
	return fmt.Sprintf("%s/%s/sensors/%d/availability", baseTopic, a.Ctl.GetIdentifier(), a.ID)
}

func (a *LeakoSensor) GetMQTTStateTopic(baseTopic string) string {
	return fmt.Sprintf("%s/%s/sensors/%d/state", baseTopic, a.Ctl.GetIdentifier(), a.ID)
}

func (a *LeakoSensor) GetMQTTLeakHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/sensor-%d-leak/config", hassPrefix, a.Ctl.GetMQTTHassNodeId(), a.ID)
}

func (a *LeakoSensor) GetMQTTLeakHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              fmt.Sprintf("%d", a.ID),
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "moisture",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_%d-leak", a.Ctl.GetMQTTHassNodeId(), a.ID),
		ValueTemplate:     "{{ value_json.leak | abs }}",
		PayloadOff:        "0",
		PayloadOn:         "1",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.Ctl.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.Ctl.GetMQTTHassNodeId()},
		},
	}
}

func (a *LeakoSensor) GetMQTTBatHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/sensor-%d-bat/config", hassPrefix, a.Ctl.GetMQTTHassNodeId(), a.ID)
}

func (a *LeakoSensor) GetMQTTBatHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              fmt.Sprintf("%d", a.ID),
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "battery",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_%d-battery", a.Ctl.GetMQTTHassNodeId(), a.ID),
		ValueTemplate:     "{{ value_json.low_bat | abs }}",
		PayloadOff:        "0",
		PayloadOn:         "1",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.Ctl.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.Ctl.GetMQTTHassNodeId()},
		},
	}
}

func (a *LeakoSensor) GetMQTTSignalHassConfigTopic(hassPrefix string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/sensor-%d-lost/config", hassPrefix, a.Ctl.GetMQTTHassNodeId(), a.ID)
}

func (a *LeakoSensor) GetMQTTSignalHassConfig(baseTopic string) *HassDiscoveryPayload {
	return &HassDiscoveryPayload{
		Name:              fmt.Sprintf("%d", a.ID),
		AvailabilityTopic: a.GetMQTTAvailabilityTopic(baseTopic),
		DeviceClass:       "connectivity",
		StateTopic:        a.GetMQTTStateTopic(baseTopic),
		UniqueId:          fmt.Sprintf("%s_%d-signal", a.Ctl.GetMQTTHassNodeId(), a.ID),
		ValueTemplate:     "{{ value_json.lost_signal | abs }}",
		PayloadOff:        "1",
		PayloadOn:         "0",
		Device: HassDeviceDiscoveryPayload{
			Name:         a.Ctl.GetMQTTHassNodeId(),
			Manufacturer: MANUFACTURER,
			Identifiers:  []string{a.Ctl.GetMQTTHassNodeId()},
		},
	}
}

func (a *LeakoSensor) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LowBat     bool `json:"low_bat"`
		LostSignal bool `json:"lost_signal"`
		Leak       bool `json:"leak"`
	}{
		LowBat:     a.IsBatLow(),
		LostSignal: a.IsLostSignal(),
		Leak:       a.IsWaterDetected(),
	})
}
