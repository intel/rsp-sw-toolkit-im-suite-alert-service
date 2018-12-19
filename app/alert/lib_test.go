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

package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

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

	os.Exit(m.Run())
}

func Test_processAlert(t *testing.T) {
	notificationChan := make(chan Notification, config.AppConfig.NotificationChanSize)
	inputData := mockGenerateAlertFromGateway()
	alertError := ProcessAlert(&inputData, notificationChan)
	config.AppConfig.AlertDestination = "http://www.test.com"
	if alertError != nil {
		t.Errorf("Error processing alerts %s", alertError)
	}
	go NotifyChannel(notificationChan)
}

func Test_processAlert_NoDestination(t *testing.T) {
	notificationChan := make(chan Notification, config.AppConfig.NotificationChanSize)
	inputData := mockGenerateAlertFromGateway()
	alertError := ProcessAlert(&inputData, notificationChan)
	config.AppConfig.AlertDestination = ""
	if alertError != nil {
		t.Errorf("Error processing alerts %s", alertError)
	}
	go NotifyChannel(notificationChan)
}

func TestGeneratePayloadAlert_withDestination(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateAlert()
	alertPayloadURL := "http://www.test.com"
	var alert models.Alert
	err := json.Unmarshal(inputData, &alert)
	if err != nil {
		t.Errorf("error parsing Alert: %s", err)
	}

	testNotification.NotificationType = "Alert"
	testNotification.NotificationMessage = "ProcessAlert"
	testNotification.Data = alert
	testNotification.GatewayID = "rrs-gateway"
	testNotification.Endpoint = alertPayloadURL
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	generateErr := testNotification.GeneratePayload()
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
		t.Error("Alert data and generated payload data is not equal")
	}

}

func TestGeneratePayloadAlert_noDestination(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateAlert()
	alertPayloadURL := ""
	var alert models.Alert
	err := json.Unmarshal(inputData, &alert)
	if err != nil {
		t.Errorf("error parsing Alert: %s", err)
	}

	testNotification.NotificationType = "Alert"
	testNotification.NotificationMessage = "ProcessAlert"
	testNotification.Data = alert
	testNotification.GatewayID = "rrs-gateway"
	testNotification.Endpoint = alertPayloadURL
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	generateErr := testNotification.GeneratePayload()
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
		t.Error("Alert data and generated payload data is not equal")
	}

}

func TestGeneratePayloadHeartbeat(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateHeartbeat()
	hbPayloadURL := config.AppConfig.HeartbeatDestination
	var hb models.Heartbeat
	err := json.Unmarshal(inputData, &hb)
	if err != nil {
		t.Errorf("error parsing Heartbeat: %s", err)
	}

	testNotification.NotificationType = models.HeartbeatType
	testNotification.NotificationMessage = "ProcessHeartbeat"
	testNotification.Data = hb
	testNotification.GatewayID = "rrs-gateway"
	testNotification.Endpoint = hbPayloadURL
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()

	generateErr := testNotification.GeneratePayload()
	if generateErr != nil {
		t.Errorf("Error in generating payload %v", generateErr)
	}
	notifyData, ok := testNotification.Data.(models.CloudConnectorPayload)
	if !ok {
		t.Error("Found incompatible payload type in notification data")
	}
	if notifyData.URL != hbPayloadURL {
		t.Error("Generated Payload has wrong URL for sending heartbeats")
	}
	heartbeatData, ok := notifyData.Payload.(models.Heartbeat)
	if !ok {
		t.Error("Body of payload is not of heartbeat type")
	}
	validData := reflect.DeepEqual(heartbeatData, hb)
	if !validData {
		t.Error("Heartbeat data and generated payload data is not equal")
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
	_, postErr := PostNotification(inputData, mockCloudConnector)
	if postErr != nil {
		t.Errorf("Posting notification failed %s", postErr)
	}
	mockCloudConnector = "http://wrongURL:8080" + "/aws-test/invoke"
	_, postErr = PostNotification(inputData, mockCloudConnector)
	if postErr == nil {
		t.Error("Posting notification was successful with wrong URL")
	}

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
			"optional": {}
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

func getTestMockServer() (*httptest.Server, error) {
	var serverErr error
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			serverErr = errors.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}
		data := "success"
		jsonData, _ := json.Marshal(data)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)

	}))
	return testServer, serverErr
}
