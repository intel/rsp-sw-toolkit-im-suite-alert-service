/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package models

import (
	"time"

	"github.impcloud.net/Responsive-Retail-Core/utilities/helper"
)

// Alert value for cloud which does not include gateway_id
type Alert struct {
	SentOn           int64       `json:"sent_on"`
	Facilities       []string    `json:"facilities"`
	DeviceID         string      `json:"device_id"`
	AlertNumber      int         `json:"alert_number"`
	AlertDescription string      `json:"alert_description"`
	Severity         string      `json:"severity"`
	Optional         interface{} `json:"optional"`
}

// AlertMessage is the data from Context sensing SDK
type AlertMessage struct {
	MACAddress  string    `json:"macaddress"`
	Application string    `json:"application"`
	ProviderID  int       `json:"providerId"`
	Datetime    time.Time `json:"dateTime, string"`
	Value       Alert     `json:"value"`
}

// GatewayRegisteredAlert generated when a new gateway is seen in a heartbeat
func GatewayRegisteredAlert(heartbeat Heartbeat) (Alert, string) {
	var register Alert
	optionalMap := make(map[string]interface{})

	register.AlertNumber = 320
	register.AlertDescription = "Gateway " + heartbeat.DeviceID + " registered"
	register.Severity = "info"

	register.SentOn = helper.UnixMilliNow()
	register.Facilities = heartbeat.Facilities
	register.DeviceID = heartbeat.DeviceID
	if heartbeat.MeshID != "" {
		optionalMap["mesh_id"] = heartbeat.MeshID
	}
	if heartbeat.MeshNodeID != "" {
		optionalMap["mesh_node_id"] = heartbeat.MeshNodeID
	}
	register.Optional = optionalMap
	return register, heartbeat.DeviceID
}

// GatewayDeregisteredAlert generated when maximum number of gateway heartbeats are missed
func GatewayDeregisteredAlert(heartbeat Heartbeat) (Alert, string) {
	var deregister Alert
	optionalMap := make(map[string]interface{})

	deregister.AlertNumber = 322
	deregister.AlertDescription = "Gateway " + heartbeat.DeviceID + " deregistered"
	deregister.Severity = "urgent"

	deregister.SentOn = helper.UnixMilliNow()
	deregister.Facilities = heartbeat.Facilities
	deregister.DeviceID = heartbeat.DeviceID
	if heartbeat.MeshID != "" {
		optionalMap["mesh_id"] = heartbeat.MeshID
	}
	if heartbeat.MeshNodeID != "" {
		optionalMap["mesh_node_id"] = heartbeat.MeshNodeID
	}
	deregister.Optional = optionalMap

	return deregister, heartbeat.DeviceID
}

// GatewayMissedHeartbeatAlert generated when a gateway heartbeat is missed
func GatewayMissedHeartbeatAlert(heartbeat Heartbeat) (Alert, string) {
	var heartbeatMissed Alert
	optionalMap := make(map[string]interface{})

	heartbeatMissed.AlertNumber = 321
	heartbeatMissed.AlertDescription = "Gateway " + heartbeat.DeviceID + " missed heartbeat"
	heartbeatMissed.Severity = "critical"

	heartbeatMissed.SentOn = helper.UnixMilliNow()
	heartbeatMissed.Facilities = heartbeat.Facilities
	heartbeatMissed.DeviceID = heartbeat.DeviceID
	if heartbeat.MeshID != "" {
		optionalMap["mesh_id"] = heartbeat.MeshID
	}
	if heartbeat.MeshNodeID != "" {
		optionalMap["mesh_node_id"] = heartbeat.MeshNodeID
	}

	heartbeatMissed.Optional = optionalMap

	return heartbeatMissed, heartbeat.DeviceID
}
