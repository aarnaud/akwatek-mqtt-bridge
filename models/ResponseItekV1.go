package models

type ResBodyItekV1 struct {
	ItekV1 ResItekV1 `json:"Itek_V1"`
}

type ResItekV1 struct {
	Message string       `json:"mess"`
	Valve   *ValveAction `json:"valve,omitempty"`
}
