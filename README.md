# Intel® Inventory Suite rfid-alert-service
[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)

RFID Alert service provides the capabilities to monitor Intel® RSP Controller status and send alerts. Intel® RSP Controller status is updated periodically based on events sent through Heartbeat listener from EdgeX Core Data and alerts are generated when heartbeats are missed. Also, alert events are processed and posted to the REST endpoint specified. 

[Need more explanation for ASN]

# Depends on

- Cloud-connector
- Product-data-service
- EdgeX Core-data

# Install and Deploy via Docker Container #

### Prerequisites ###

Intel® RSP Software Toolkit 

- [RSP Controller](https://github.com/intel/rsp-sw-toolkit-gw)
- [RSP MQTT Device Service](https://github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service)

EdgeX and RSP MQTT Device Service should be running at this point.

### Installation ###

```
git clone https://github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service.git
cd rfid-alert-service
sudo make build
sudo make deploy
```


