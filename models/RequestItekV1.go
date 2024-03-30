package models

import (
	"encoding/json"
	"net"
	"strings"
)

type ReqBodyItekV1 struct {
	ItekV1 ReqItekV1 `json:"Itek_V1"`
}

type ReqItekV1 struct {
	MacAddress  net.HardwareAddr `json:"MAC_address"`
	ID          string           `json:"ID"`
	CtlStatus   string           `json:"Cont_status"`
	Zone01To25  string           `json:"zone01-25"`
	Zone26To50  string           `json:"zone26-50"`
	Zone51To75  string           `json:"zone51-75"`
	Zone76To100 string           `json:"zone76-100"`
}

func (i *ReqItekV1) GetIdentifier() string {
	return strings.ReplaceAll(i.MacAddress.String(), ":", "")
}

func (i *ReqItekV1) UnmarshalJSON(data []byte) error {
	var err error
	type Alias ReqItekV1
	aux := &struct {
		MacAddress string `json:"MAC_address"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.MacAddress, err = net.ParseMAC(aux.MacAddress)
	if err != nil {
		return err
	}
	return nil
}
