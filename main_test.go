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
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
)

func TestMain(m *testing.M) {
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	notificationChan = make(chan Notification, config.AppConfig.NotificationChanSize)
	os.Exit(m.Run())
}

func Test_processAlert(t *testing.T) {
	inputData := mockGenerateAlertFromGateway()
	alertError := processAlert(&inputData)
	if alertError != nil {
		t.Errorf("Error processing alerts %s", alertError)
	}
}

func TestProcessHeartbeat(t *testing.T) {
	inputData := mockGenerateHeartbeat()
	heartBeatError := processHeartbeat(&inputData)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}

}

func TestGeneratePayloadAlert(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateAlert()
	alertPayloadURL := config.AppConfig.AlertDestination
	var alert models.Alert
	err := json.Unmarshal(inputData, &alert)
	if err != nil {
		t.Errorf("error parsing Heartbeat: %s", err)
	}

	testNotification.NotificationType = "Alert"
	testNotification.NotificationMessage = "ProcessAlert"
	testNotification.Data = alert
	testNotification.GatewayID = "rrs-gateway"
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	generateErr := testNotification.generatePayload()
	if generateErr != nil {
		t.Errorf("Error in generating payload %v", generateErr)
	}
	notifyData, ok := testNotification.Data.(models.CloudConnectorPayload)
	if !ok {
		t.Error("Found incompatible payload type in notification data")
	}
	if notifyData.URL != alertPayloadURL {
		t.Error("Generated Payload has wrong URL for sending alerts")
	}
	alertData, ok := notifyData.Payload.(models.Alert)
	if !ok {
		t.Error("Body of payload is not of alert type")
	}
	validData := reflect.DeepEqual(alertData, alert)
	if !validData {
		t.Error("Heartbeat data and generated payload data is not equal")
	}

}

func TestGatewayStatus(t *testing.T) {
	watchdogSeconds := 1
	//Starting gateway status check in separate goroutine
	go monitorHeartbeat(watchdogSeconds)
	missedHeartBeats := config.AppConfig.MaxMissedHeartbeats

	// check for gateway registered alert
	inputData := mockGenerateHeartbeat()
	heartBeatError := processHeartbeat(&inputData)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}
	if gateway.RegistrationStatus != models.Registered {
		t.Error("Failed to register gateway")
	}


	//Delay heartbeat by 3 seconds to check the functionality of missed heartbeat and gateway deregistered alert
	for i := 0; i <= missedHeartBeats; i++ {
		heartBeatError := processHeartbeat(&inputData)
		if heartBeatError != nil {
			t.Errorf("Error processing heartbeat %s", heartBeatError)
		}
		time.Sleep(2 * time.Second)
	}
	// looping through notification channel to make sure we get Gateway deregistred alert before checking for error conditions
	for noti := range notificationChan {
		if noti.NotificationMessage == "Gateway Deregistered Alert" {
			break
		}
	}
	if gateway.MissedHeartBeats != missedHeartBeats {
		t.Error("Failed to register missed heartbeats")
	}
	if gateway.RegistrationStatus != models.Deregistered {
		t.Error("Failed to deregister gateway")
	}
}

func TestHeartbeatAlert(t *testing.T) {
	input := mockGenerateHeartbeat()
	heartbeat, err := generateHeartbeatModel(input)
	if err != nil {
		t.Fatalf("Error generating heartbeat %s", err)
	}
	heartbeatAlert, deviceId := models.GatewayRegisteredAlert(heartbeat)
	if deviceId != heartbeat.DeviceID {
		t.Error("Alert device id does not match hearbeat device id")
	}
	if len(heartbeatAlert.Facilities) != len(heartbeat.Facilities) {
		t.Error("Number of alert facilitites does not match heartbeat facilities")
	}
	var heartbeatFacilities []string
	for _, value := range heartbeat.Facilities {
		heartbeatFacilities = append(heartbeatFacilities, value)
	}
	if !reflect.DeepEqual(heartbeatAlert.Facilities, heartbeatFacilities) {
		t.Error("Facilites from alert is not the same as heartbeat facilities")
	}


	input = mockGenerateHeartbeatNoFacility()
	heartbeat, err = generateHeartbeatModel(input)
	if err != nil {
		t.Fatalf("Error generating heartbeat with no facility %s", err)
	}
	heartbeatAlert, deviceId = models.GatewayRegisteredAlert(heartbeat)
	// Alert generated from heartbeat with no facilities should have facilities field with value "UNDEFINED_FACILITY"
	if len(heartbeatAlert.Facilities) != 1 {
		t.Errorf("Alert generated from heartbeat with no facilities should have a length of one")
	}
	if heartbeatAlert.Facilities[0] != models.UndefinedFacility {
		t.Errorf("Alert generated from heartbeat with no facilities has the wrong facility defined")
	}
}


func TestPostNotification(t *testing.T) {
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()
	inputData := mockGenerateHeartbeat()
	mockCloudConnector := testMockServer.URL + "/aws-test/invoke"
	_, postErr := postNotification(inputData, mockCloudConnector)
	if postErr != nil {
		t.Errorf("Posting notification failed %s", postErr)
	}
	mockCloudConnector = "http://wrongURL:8080" + "/aws-test/invoke"
	_, postErr = postNotification(inputData, mockCloudConnector)
	if postErr == nil {
		t.Error("Posting notification was successful with wrong URL")
	}

}

func getTestMockServer() (*httptest.Server, error) {
	var serverErr error
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			serverErr = errors.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}
		switch request.URL.EscapedPath() {

		case "/	aws/invoke":
			data := "success"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		}
	}))
	return testServer, serverErr
}

// Alert from gateway which includes gateway_id field
func mockGenerateAlertFromGateway() []byte {
	testAlert := []byte(`{
			"macaddress":  "02:42:ac:1a:00:05",
			"application": "rsp_collector-service",
			"providerId":  -1,
  			"dateTime":    "2018-04-13T20:03:11.328Z",
			"value": {
						"sent_on": 1523904547000,
						"facilities": ["front"],
						"device_id": "Sensor1",
						"gateway_id": "rrs-gateway",
						"alert_number": 22,
						"alert_description": "sensor disconnected",
						"severity": "info"
			}
	}`)
	return testAlert
}

// Alert for cloud which excludes gateway_id field
func mockGenerateAlert() []byte {
	alert := []byte(`{
		"macaddress":  "02:42:ac:1a:00:05",
		"application": "rsp_collector-service",
		"providerId":  -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"value": {
			"sent_on": 1503700192960,
			"facilities": ["front"],
			"device_id": "Sensor1",
			"alert_number": 22,
			"alert_description": "sensor disconnected",
			"severity": "info", 
			"optional": { "mesh_id": "rrs-gateway" }
		}
	}`)
	return alert
}

func mockGenerateHeartbeat() []byte {
	heartbeat := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
		"value": {
		  "device_id": "rrpgw",
		  "gateway_id": "rrpgw",
		  "facilities": [
				"facility1",
				"facility2"
		  ],
		  "facility_groups_cfg": "auto-0802233641",
		  "mesh_id": null,
		  "mesh_node_id": null,
		  "personality_groups_cfg": null,
		  "schedule_cfg": "UNKNOWN",
		  "schedule_groups_cfg": null,
		  "sent_on": 1503700192960
		}
	}`)
	return heartbeat
}

func mockGenerateHeartbeatNoFacility() []byte {
	heartbeat := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
		"value": {
		  "device_id": "rrpgw",
		  "gateway_id": "rrpgw",
		  "facilities": [],
		  "facility_groups_cfg": null,
		  "mesh_id": null,
		  "mesh_node_id": null,
		  "personality_groups_cfg": null,
		  "schedule_cfg": "UNKNOWN",
		  "schedule_groups_cfg": null,
		  "sent_on": 1503700192960
		}
	}`)
	return heartbeat
}

func generateHeartbeatModel(input []byte) (models.Heartbeat, error) {
	var heartbeatEvent models.HeartbeatMessage
	err := json.Unmarshal(input, &heartbeatEvent)
	if err != nil {
		log.Fatalf("error parsing Heartbeat: %s", err)
		return heartbeatEvent.Value, err
	}

	return heartbeatEvent.Value, nil
}