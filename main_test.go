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

	//Setting mock AWS values for testing
	copyConfig := config.AppConfig
	config.AppConfig.CloudConnectorURL = "https://mockcloudconnector:8080/"
	config.AppConfig.CloudConnectorEndpoint = "/aws/invoke"
	notificationChan = make(chan Notification, config.AppConfig.NotificationChanSize)
	os.Exit(m.Run())
	//Setting old config values back
	config.AppConfig = copyConfig
}

func Test_processAlert(t *testing.T) {
	inputData := mockGenerateAlertFromGateway()
	alertError := processAlert(inputData)
	if alertError != nil {
		t.Errorf("Error processing alerts %s", alertError)
	}
}

func TestProcessHeartbeat(t *testing.T) {
	inputData := mockGenerateHeartBeats()
	heartBeatError := processHeartbeat(inputData)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}

}

func TestGeneratePayloadHeartBeat(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateHeartBeats()
	heartbeatPayloadURL := "https://" + config.AppConfig.AwsURLHost + config.AppConfig.AwsURLStage + config.AppConfig.HeartbeatEndpoint
	var hb models.HeartbeatMessage
	err := json.Unmarshal(*inputData, &hb)
	if err != nil {
		t.Errorf("error parsing Heartbeat: %s", err)
	}

	testNotification.NotificationType = "HeartBeat"
	testNotification.NotificationMessage = "ProcessHeartbeat"
	testNotification.Data = hb.Value
	testNotification.GatewayID = hb.Value.DeviceID

	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()
	config.AppConfig.JwtSignerURL = testMockServer.URL

	generateErr := testNotification.generatePayload()
	if generateErr != nil {
		t.Errorf("Error in generating payload %v", generateErr)
	}
	notifyData, ok := testNotification.Data.(models.CloudConnectorPayload)
	if !ok {
		t.Error("Found incompatible payload type in notification data")
	}
	if notifyData.URL != heartbeatPayloadURL {
		t.Error("Generated Payload has wrong URL for sending heartbeats")
	}
	hbData, ok := notifyData.Payload.(models.Heartbeat)
	if !ok {
		t.Error("Body of payload is not of heartbeat type")
	}
	validData := reflect.DeepEqual(hbData, hb.Value)
	if !validData {
		t.Error("Heartbeat data and generated payload data is not equal")
	}
}

func TestGeneratePayloadAlert(t *testing.T) {
	testNotification := new(Notification)
	inputData := mockGenerateAlert()
	alertPayloadURL := "https://" + config.AppConfig.AwsURLHost + config.AppConfig.AwsURLStage + config.AppConfig.AlertEndpoint
	var alert models.Alert
	err := json.Unmarshal(*inputData, &alert)
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
	config.AppConfig.JwtSignerURL = testMockServer.URL

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
	inputData := mockGenerateHeartBeats()
	heartBeatError := processHeartbeat(inputData)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}
	if gateway.RegistrationStatus != models.Registered {
		t.Error("Failed to register gateway")
	}


	//Delay heartbeat by 3 seconds to check the functionality of missed heartbeat and gateway deregistered alert
	for i := 0; i <= missedHeartBeats; i++ {
		heartBeatError := processHeartbeat(inputData)
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

func TestPostNotification(t *testing.T) {
	testMockServer, serverErr := getTestMockServer()
	if serverErr != nil {
		t.Errorf("Server returned a error %v", serverErr)
	}
	defer testMockServer.Close()
	inputData := mockGenerateHeartBeats()
	mockCloudConnector := testMockServer.URL + "/aws-test/invoke"
	_, postErr := postNotification(inputData, mockCloudConnector)
	if postErr != nil {
		t.Errorf("Posting notification failed %s", postErr)
	}
	mockCloudConnector = testMockServer.URL + "/jwt-signing/sign"
	jwtResponse, postErr := postNotification(inputData, mockCloudConnector)
	if postErr != nil {
		t.Errorf("Posting notification failed %s", postErr)
	}
	if jwtResponse == nil {
		t.Errorf("JWTResponse is nil %s", postErr)
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
		case "/jwt-signing/sign":
			data := "xxxxx.yyyyy.zzzzz"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)

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
func mockGenerateAlertFromGateway() *[]byte {
	testAlert := []byte(
		`{
			"macaddress":  "02:42:ac:1a:00:05",
			"application": "rsp_collector-service",
			"providerId":  -1,
  			"dateTime":    "2018-04-13T20:03:11.328Z",
			"value": 	   {
					 		 "sent_on": 		  1523904547000,
							 "facilities": 		  ["front"],
 				     		 "device_id":         "Sensor1",
				     		 "gateway_id":		  "rrs-gateway",
					 		 "alert_number":      22,
					 		 "alert_description": "sensor disconnected",
					 		 "severity":          "info"
			}
		}`)
	return &testAlert
}

// Alert for cloud which excludes gateway_id field
func mockGenerateAlert() *[]byte {
	currentTime := time.Now()
	testAlert := models.Alert{
		SentOn:           currentTime.AddDate(0, 0, -1).Unix(),
		Facilities:       []string{"front"},
		DeviceID:         "Sensor1",
		AlertNumber:      22,
		AlertDescription: "sensor disconnected",
		Severity:         "info",
	}

	testAlertMessage := models.AlertMessage{
		MACAddress:  "02:42:ac:1a:00:05",
		Application: "rsp_collector-service",
		ProviderID:  -1,
		Datetime:    time.Now(),
		Value:     testAlert,
	}
	inputData, _ := json.Marshal(testAlertMessage)
	return &inputData
}

func mockGenerateHeartBeats() *[]byte {
	testHeartBeatMessageValue := models.Heartbeat{
		Facilities:           []string{"front"},
		FacilityGroupsCfg:    "auto-0310080051",
		MeshID:               "MeshID78",
		MeshNodeID:           "MeshNodeID78",
		PersonalityGroupsCfg: "test",
		ScheduleCfg:          "test",
		ScheduleGroupsCfg:    "test",
		SentOn:               1506532617643,
	}
	testHeartBeat := models.HeartbeatMessage{
		MACAddress:  "02:42:ac:1a:00:05",
		Application: "rsp_collector-service",
		ProviderID:  -1,
		Datetime:    time.Now(),
		Value:     testHeartBeatMessageValue,
	}
	inputData, _ := json.Marshal(testHeartBeat)
	return &inputData
}
