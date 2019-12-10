/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package models

import (
	"time"
)

const (
	// HeartbeatType is the notification type for a heartbeat
	HeartbeatType = "Heartbeat"
)

// Heartbeat from gateway
type Heartbeat struct {
	// DeviceID is gateway id
	DeviceID             string   `json:"device_id"`
	Facilities           []string `json:"facilities"`
	FacilityGroupsCfg    string   `json:"facility_groups_cfg"`
	MeshID               string   `json:"mesh_id"`
	MeshNodeID           string   `json:"mesh_node_id"`
	PersonalityGroupsCfg string   `json:"personality_groups_cfg"`
	ScheduleCfg          string   `json:"schedule_cfg"`
	ScheduleGroupsCfg    string   `json:"schedule_groups_cfg"`
	SentOn               int      `json:"sent_on"`
}

// Heartbeat message from SAF
type HeartbeatMessage struct {
	MACAddress  string    `json:"macaddress"`
	Application string    `json:"application"`
	ProviderID  int       `json:"providerId"`
	Datetime    time.Time `json:"dateTime,string"`
	Value       Heartbeat `json:"value"`
}
