# Intel® Inventory Suite rfid-alert-service
[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)

The RFID Alert Service generates and routes alerts for Intel® RSP Controller
status, Advance Shipping Notices, and upstream service alerts.

- *Intel® RSP Controller Status*: 
    the service listens to EdgeX Core Data for controller heartbeats and,
    if it misses multiple consecutive heartbeats, it sends an alert.
- *ASN Events*:
    when enabled, the service verifies Advance Shipping Notices' products' 
    statuses and sends alerts for any that aren't whitelisted.
- *Upstream Alerts*:
    the service processes incoming alerts from other services and posts them to 
    a configured REST endpoint. 

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
make build deploy
```

### API Documentation ###

Go to [https://editor.swagger.io](https://editor.swagger.io) and import rfid-alert-service.yml file.
