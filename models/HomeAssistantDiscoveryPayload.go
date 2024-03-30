package models

import "encoding/json"

type HassDeviceDiscoveryPayload struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Model        string   `json:"model,omitempty"`
	SerialNumber string   `json:"serial_number,omitempty"`
	HwVersion    string   `json:"hw_version,omitempty"`
	SwVersion    string   `json:"sw_version,omitempty"`
}

type HassDiscoveryPayload struct {
	Name              string                     `json:"name"`
	DeviceClass       string                     `json:"device_class"`
	StateTopic        string                     `json:"state_topic"`
	CommandTopic      string                     `json:"command_topic,omitempty"`
	AvailabilityTopic string                     `json:"availability_topic,omitempty"`
	UniqueId          string                     `json:"unique_id"`
	UnitOfMeasurement string                     `json:"unit_of_measurement,omitempty"`
	ValueTemplate     string                     `json:"value_template,omitempty"`
	PayloadOff        string                     `json:"payload_off,omitempty"`
	PayloadOn         string                     `json:"payload_on,omitempty"`
	ReportsPosition   bool                       `json:"reports_position,omitempty"`
	Device            HassDeviceDiscoveryPayload `json:"device"`
}

func (h *HassDiscoveryPayload) MarshalJSON() ([]byte, error) {
	type Alias HassDiscoveryPayload
	alias := (*Alias)(h)

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: alias,
	})
}
