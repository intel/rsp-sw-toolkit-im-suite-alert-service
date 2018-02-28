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

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
)

type inputTest struct {
	title string
	input []byte
}

func TestMain(m *testing.M) {
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}
	os.Exit(m.Run())
}

func Test_processAlert(t *testing.T) {
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	inputData := mockGenerateAlerts()
	alertError := processAlert(inputData)
	if alertError != nil {
		t.Errorf("Error processing alerts %s", alertError)
	}
}

func Test_processHeartbeat(t *testing.T) {
	watchdogSeconds := 1
	go initGatewayStatusCheck(watchdogSeconds)
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	inputData := mockGenerateHeartBeats()
	heartBeatError := processHeartbeat(inputData)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}

}

func TestGatewayDeregister(t *testing.T) {
	watchdogSeconds := 1
	//Starting gateway status check in separate goroutine
	go initGatewayStatusCheck(watchdogSeconds)
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	missedHeartBeats := config.AppConfig.MaxMissedHeartbeats

	//Delay heartbeat by 3 seconds to check the functionality of missed heartbeat and gateway deregistered alert
	inputData := mockGenerateHeartBeats()
	for i := 0; i <= missedHeartBeats; i++ {
		heartBeatError := processHeartbeat(inputData)
		if heartBeatError != nil {
			t.Errorf("Error processing heartbeat %s", heartBeatError)
		}
		time.Sleep(3 * time.Second)
	}
	if gw.MissedHeartBeats != missedHeartBeats {
		t.Error("Failed to register missed heartbeats")
	}
	if gw.RegistrationStatus != models.Deregistered {
		t.Error("Failed to deregister gateway")
	}

}

func TestHeartBeatMessageValidateSchemaRequest(t *testing.T) {

	var invalidJSONSample = []inputTest{
		{
			title: "Required field sent_on missing",
			input: []byte(`{
				"macaddress": "02:42:ac:1a:00:05",
				"application": "rsp_collector",
				"providerId": -1,
				"dateTime": "2017-09-27T17:16:57.644Z",
				"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
				"value": {
				  "device_id": "rsdrrp",
				  "facilities": [
					"front"
				  ],
				  "facility_groups_cfg": "auto-0310080051",
				  "mesh_id": null,
				  "mesh_node_id": null,
				  "personality_groups_cfg": null,
				  "schedule_cfg": "UNKNOWN",
				  "schedule_groups_cfg": null
				}
			  }`),
		},
		{
			title: "datetime field invalid format",
			input: []byte(`{
			"macaddress": "02:42:ac:1a:00:05",
			"application": "rsp_collector",
			"providerId": -1,
			"dateTime": "test",
			"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
			"value": {
			  "device_id": "rsdrrp",
			  "facilities": [
				"front"
			  ],
			  "facility_groups_cfg": "auto-0310080051",
			  "mesh_id": null,
			  "mesh_node_id": null,
			  "personality_groups_cfg": null,
			  "schedule_cfg": "UNKNOWN",
			  "schedule_groups_cfg": null,
			  "sent_on": 1506532617643
			}
		  }`),
		},
		{
			// Empty request body
			title: "Empty request body",
			input: []byte(`{}`),
		},
	}
	for _, item := range invalidJSONSample {
		validErr := ValidateSchemaRequest(item.input, models.HeartBeatSchema)
		if validErr == nil {
			t.Errorf("Schema validation should have failed due to this reason %s", item.title)
		}
	}

}

func TestAlertValidatechemaRequest(t *testing.T) {

	var invalidJSONSample = []inputTest{
		{
			title: "Required field sent_on missing",
			input: []byte(`{
				"facilities":["front"],
				"device_id":"test",
				"alert_number":1234,
				"alert_description":"Sensor",
				"severity": "info",
			  }`),
		},
		{
			title: "Wrong severity type",
			input: []byte(`{
				"sent_on": 1506532617643
				"facilities":["front"],
				"device_id":"test",
				"alert_number":1234,
				"alert_description":"Sensor",
				"severity": "inf",
			  }`),
		},
		{
			title: "Empty request body",
			input: []byte(`{}`),
		},
	}
	for _, item := range invalidJSONSample {
		validErr := ValidateSchemaRequest(item.input, models.HeartBeatSchema)
		if validErr == nil {
			t.Errorf("Schema validation should have failed due to this reason %s", item.title)
		}
	}

}

func getTestMockServer() (*httptest.Server, error) {
	var serverErr error
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			serverErr = errors.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}
		switch request.URL.EscapedPath() {
		case "/alert":
			data := "received alert"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		case "/heartbeat":
			data := "received heartbeat"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		}
	}))
	config.AppConfig.SendAlertTo = testServer.URL + "/alert"
	config.AppConfig.SendHeartbeatTo = testServer.URL + "/hearbeat"
	return testServer, serverErr
}

func mockGenerateAlerts() *[]byte {
	currentTime := time.Now()
	testAlert := models.Alert{
		SentOn:           currentTime.AddDate(0, 0, -1).Unix(),
		Facilities:       []string{"front"},
		DeviceID:         "Sensor1",
		AlertNumber:      22,
		AlertDescription: "sensor disconnected",
		Severity:         "info",
	}
	inputData, _ := json.Marshal(testAlert)
	return &inputData
}

func mockGenerateHeartBeats() *[]byte {
	testHeartBeatMessageValue := models.HeartBeatMessageValue{
		DeviceID:             "rsdrrp",
		Facilities:           []string{"front"},
		FacilityGroupsCfg:    "auto-0310080051",
		MeshID:               "MeshID78",
		MeshNodeID:           "MeshNodeID78",
		PersonalityGroupsCfg: "test",
		ScheduleCfg:          "test",
		ScheduleGroupsCfg:    "test",
		SentOn:               1506532617643,
	}
	testHeartBeat := models.HeartBeatMessage{
		MACAddress:  "02:42:ac:1a:00:05",
		Application: "rsp_collector-service",
		ProviderID:  -1,
		Datetime:    time.Now(),
		Details:     testHeartBeatMessageValue,
	}
	inputData, _ := json.Marshal(testHeartBeat)
	return &inputData
}
