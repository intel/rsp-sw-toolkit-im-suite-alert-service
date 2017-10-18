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
	"github.impcloud.net/Responsive-Retail-MVP/utilities/helper"
)

// Alert is the base data that is provided in an alert
type Alert struct {
	SentOn           int64
	DeviceID         string
	AlertNumber      int
	AlertDescription string
	Severity         string
	MeshID           string
	MeshNodeID       string
}

// GatewayAlert is the base data that is provided in a Gateway Alert
type GatewayAlert struct {
	Facilities []string
	Alert
}

// GatewayRegisteredAlert is sent when a gateway is registered
type GatewayRegisteredAlert struct {
	GatewayAlert
}

// GatewayDeregisteredAlert is sent when a gateway is de-registered
type GatewayDeregisteredAlert struct {
	GatewayAlert
}

// GatewayMissedHeartbeatAlert is sent when a gateway has missed a heart beat
type GatewayMissedHeartbeatAlert struct {
	GatewayAlert
}

// NewGatewayRegisteredAlert will populate a
// GatewayRegisteredAlert given a HeartBeatMessage
func NewGatewayRegisteredAlert(hb HeartBeatMessage) GatewayRegisteredAlert {
	var gwm GatewayRegisteredAlert

	gwm.AlertNumber = 320
	gwm.AlertDescription = "Gateway " + hb.Details.DeviceID + " registered"
	gwm.Severity = "info"

	gwm.SentOn = helper.UnixMilliNow()
	gwm.Facilities = hb.Details.Facilities
	gwm.DeviceID = hb.Details.DeviceID
	gwm.MeshID = hb.Details.MeshID
	gwm.MeshNodeID = hb.Details.MeshNodeID

	return gwm
}

// NewGatewayDeregisteredAlert will populate a
// GatewayDeregisteredAlert given a HeartBeatMessage
func NewGatewayDeregisteredAlert(hb HeartBeatMessage) GatewayDeregisteredAlert {
	var gwm GatewayDeregisteredAlert

	gwm.AlertNumber = 322
	gwm.AlertDescription = "Gateway " + hb.Details.DeviceID + " deregistered"
	gwm.Severity = "urgent"

	gwm.SentOn = helper.UnixMilliNow()
	gwm.Facilities = hb.Details.Facilities
	gwm.DeviceID = hb.Details.DeviceID
	gwm.MeshID = hb.Details.MeshID
	gwm.MeshNodeID = hb.Details.MeshNodeID

	return gwm
}

// NewGatewayMissedHeartbeatAlert will populate a
// GatewayMissedHeartbeatAlert given a HeartBeatMessage
func NewGatewayMissedHeartbeatAlert(hb HeartBeatMessage) GatewayMissedHeartbeatAlert {
	var gwm GatewayMissedHeartbeatAlert

	gwm.AlertNumber = 321
	gwm.AlertDescription = "Gateway " + hb.Details.DeviceID + " missed heartbeat"
	gwm.Severity = "critical"

	gwm.SentOn = helper.UnixMilliNow()
	gwm.Facilities = hb.Details.Facilities
	gwm.DeviceID = hb.Details.DeviceID
	gwm.MeshID = hb.Details.MeshID
	gwm.MeshNodeID = hb.Details.MeshNodeID

	return gwm
}
