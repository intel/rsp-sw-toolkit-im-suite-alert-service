/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package models

import (
	"time"

	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
)

type Alert struct {
	SentOn     int64    `json:"sent_on"`
	Facilities []string `json:"facilities"`
	// DeviceID is sensor id
	DeviceID         string      `json:"device_id"`
	AlertNumber      int         `json:"alert_number"`
	AlertDescription string      `json:"alert_description"`
	Severity         string      `json:"severity"`
	ControllerID     string      `json:"controller_id"`
	Optional         interface{} `json:"optional"`
}

// Alert message from SAF
type AlertMessage struct {
	MACAddress  string    `json:"macaddress"`
	Application string    `json:"application"`
	ProviderID  int       `json:"providerId"`
	Datetime    time.Time `json:"dateTime,string"`
	Value       Alert     `json:"value"`
}

// UndefinedFacility is the place holder value for unused facility
const UndefinedFacility = "UNDEFINED_FACILITY"

// GatewayRegisteredAlert generated when a new gateway is seen in a heartbeat
func GatewayRegisteredAlert(heartbeat Heartbeat) (Alert, string) {
	var register Alert

	register.AlertNumber = 320
	register.AlertDescription = "Gateway " + heartbeat.DeviceID + " registered"
	register.Severity = "info"
	register.SentOn = helper.UnixMilliNow()
	register.Facilities = defineFacilities(heartbeat, register)
	register.ControllerID = heartbeat.DeviceID
	// DeviceId is same as GatewayId as there is no sensor id
	// available in a heartbeat
	register.DeviceID = heartbeat.DeviceID

	return register, heartbeat.DeviceID
}

// GatewayDeregisteredAlert generated when maximum number of gateway heartbeats are missed
func GatewayDeregisteredAlert(heartbeat Heartbeat) (Alert, string) {
	var deregister Alert

	deregister.AlertNumber = 322
	deregister.AlertDescription = "Gateway " + heartbeat.DeviceID + " deregistered"
	deregister.Severity = "urgent"

	deregister.SentOn = helper.UnixMilliNow()
	deregister.Facilities = defineFacilities(heartbeat, deregister)
	deregister.ControllerID = heartbeat.DeviceID
	// DeviceId is same as GatewayDeviceId as there is no sensor id
	// available in a heartbeat
	deregister.DeviceID = heartbeat.DeviceID

	return deregister, heartbeat.DeviceID
}

// GatewayMissedHeartbeatAlert generated when a gateway heartbeat is missed
func GatewayMissedHeartbeatAlert(heartbeat Heartbeat) (Alert, string) {
	var heartbeatMissed Alert

	heartbeatMissed.AlertNumber = 321
	heartbeatMissed.AlertDescription = "Gateway " + heartbeat.DeviceID + " missed heartbeat"
	heartbeatMissed.Severity = "critical"
	heartbeatMissed.SentOn = helper.UnixMilliNow()
	heartbeatMissed.Facilities = defineFacilities(heartbeat, heartbeatMissed)
	heartbeatMissed.ControllerID = heartbeat.DeviceID
	// DeviceId is same as GatewayDeviceId as there is no sensor id
	// available in a heartbeat
	heartbeatMissed.DeviceID = heartbeat.DeviceID

	return heartbeatMissed, heartbeat.DeviceID
}

func defineFacilities(heartbeat Heartbeat, alert Alert) []string {
	if len(heartbeat.Facilities) > 0 {
		alert.Facilities = heartbeat.Facilities
	} else {
		// if facilities field is empty
		alert.Facilities = append(alert.Facilities, UndefinedFacility)
	}

	return alert.Facilities
}
