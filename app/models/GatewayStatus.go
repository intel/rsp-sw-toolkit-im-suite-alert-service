/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
