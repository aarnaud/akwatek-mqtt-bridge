# AKWA Technologies / Leako water leak detectors MQTT bridge for Home Assistant

> /!\ Base on reverse engineering for interoperability with Home Assistant /!\

> Every information used come from public documentation or traffic interception using `mitmproxy` on my local network

Behaviour and Assumption:
- Every minutes the controller do a Post request to https://app.akwatek.com/collect2.php
  - Controlling the valve remotely can take 1 minute to be real and another minute to have state feedback
- This domain can be changed during the wifi configuration.
- The controller validate the TLS server certificate without controlling the certificate chain
  - That means you have to present any certificate signed by a CA even a Self-signed CA, not a Self-signed server certificate

## Features

- [x] Get sensors informations (leak and low battery)
- [x] Control the valve
- [x] Home Assistant MQTT Discovery
- [ ] Passthrough mode, relay controller's calls to Akwatek Cloud

## Envs

- `AMB_TLS_PORT` default `8443`
- `AMB_MQTT_BROKER_PORT`  default `1883`
- `AMB_MQTT_CLIENT_ID`  default `akwatek`
- `AMB_MQTT_BASE_TOPIC`  default `akwatek`
- `AMB_HASS_DISCOVERY_TOPIC`  default `homeassistant`
- `AMB_MQTT_BROKER_HOST`
- `AMB_MQTT_USERNAME`
- `AMB_MQTT_PASSWORD`

## Reverse engineering
### Parsing the request payload

```json
{
    "Itek_V1": {
        "Cont_status": "18041",
        "ID": "1213",
        "MAC_address": "BC:FF:4D:XX:XX:XX",
        "zone01-25": "1000000000000010000500000",
        "zone26-50": "1100000000000000100000000",
        "zone51-75": "00000000000000000000E0000",
        "zone76-100": "0000000000000000000000000"
    }
}
```
Fields:
- `Cont_status`: hexadecimal value about the controller status
- `ID`: it's seem a checksum/crc, I didn't succeed to reverse engineering
- `MAC_address`: it's obvious, it's the mac adress of the esp32 used by the controller, it's also the identifier of the controller on app.akwatek.com
- `zone01-25`: hexadecimal value of each zone

After multiple tests with different conditions, I success to identify most of the bits useful.

**Cont_status**

Example `18041`
```text
0001 1000 0000 0100 0001
```
- first hex `0001` => Power status
- second hex `1000` => Probably something related to the controller battery state
- third hex `0000` => Alarm status, `0001` if triggered
- fourth hex `0100` => not sure
- fifth hex `0001` => valve status `0001` is open `0000` is close

**zone01-25**

each hex is a sensors, `0` means no sensors configured for this zone
example of some values
```
1 => 0001
5 => 0101
9 => 1001
D => 1101
```

- least significant bit, indicate there is en sensors configured, somethings like that
- second bit, I don't know, but there is a low temperature detection, I should be this
- third bit, Low battery status
- fourth bit, Leak detection

### Parsing the response payload

Standby response
```json
{"Itek_V1":{"mess":"OK"}}
```

Valve action response to open the valve
```json
{"Itek_V1":{"mess":"OK","valve":"1"}}
```