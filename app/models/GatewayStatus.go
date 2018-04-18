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
	"sync"
	"time"
)

type Status int

const (
	Pending Status = iota
	Registered
	Deregistered
)

var (
	gateway     *gatewayStatus
	once        sync.Once
	defaultTime time.Time
)

// gatewayStatus keeps track of the current known status for the Gateway.
type gatewayStatus struct {
	gatewayMutex       sync.RWMutex
	FirstHeartbeatSeen time.Time
	LastHeartbeatSeen  time.Time
	LastHeartbeat      Heartbeat
	MissedHeartBeats   int
	RegistrationStatus Status
}

// GetInstanceGateway registers the gateway with default values and ensures that only once instance of struct is created as it is used as a global variable.
func GetInstanceGateway() *gatewayStatus {
	once.Do(func() {
		gateway = &gatewayStatus{
			RegistrationStatus: Pending,
			MissedHeartBeats:   0,
			LastHeartbeatSeen:  defaultTime,
		}
	})

	return gateway
}

func (gateway *gatewayStatus) UpdateGatewayStatus(lastHeartBeatSeen time.Time, missedHeartBeats int, hb Heartbeat) bool {
	//Mutex for safe access of gateway
	gateway.gatewayMutex.Lock()
	gateway.LastHeartbeatSeen = lastHeartBeatSeen
	gateway.LastHeartbeat = hb
	gateway.MissedHeartBeats = missedHeartBeats
	defer gateway.gatewayMutex.Unlock()
	return true
}

func (gateway *gatewayStatus) RegisterGateway() bool {
	gateway.gatewayMutex.Lock()
	gateway.RegistrationStatus = Registered
	gateway.FirstHeartbeatSeen = gateway.LastHeartbeatSeen
	defer gateway.gatewayMutex.Unlock()
	return true
}

func (gateway *gatewayStatus) UpdateMissedHeartBeats() bool {
	gateway.gatewayMutex.Lock()
	gateway.MissedHeartBeats += 1
	defer gateway.gatewayMutex.Unlock()
	return true
}

func (gateway *gatewayStatus) GetMissedHeartBeats() int {
	gateway.gatewayMutex.Lock()
	defer gateway.gatewayMutex.Unlock()
	return gateway.MissedHeartBeats
}

func (gateway *gatewayStatus) DeregisterGateway() bool {
	gateway.gatewayMutex.Lock()
	gateway.RegistrationStatus = Deregistered
	defer gateway.gatewayMutex.Unlock()
	return true
}

func (gateway *gatewayStatus) GetRegistrationStatus() Status {
	gateway.gatewayMutex.Lock()
	defer gateway.gatewayMutex.Unlock()
	return gateway.RegistrationStatus
}


func (gateway *gatewayStatus) GetLastHeartbeatSeen() time.Time {
	gateway.gatewayMutex.Lock()
	defer gateway.gatewayMutex.Unlock()
	return gateway.LastHeartbeatSeen
}


func (gateway *gatewayStatus) GetLastHeartbeat() Heartbeat {
	gateway.gatewayMutex.Lock()
	defer gateway.gatewayMutex.Unlock()
	return gateway.LastHeartbeat
}
