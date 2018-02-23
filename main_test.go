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
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
)

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

func getTestMockServer() (*httptest.Server, error) {
	var serverErr error
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			serverErr = errors.Errorf("Expected 'POST' request, received '%s'", request.Method)
		}
		switch request.URL.EscapedPath() {
		case "/alert":
			data := "recieved alert"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		case "/heartbeat":
			data := "recieved heartbeat"
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		}
	}))

	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}
	AppConfig := config.AppConfig
	AppConfig.SendAlertTo = testServer.URL + "/alert"
	AppConfig.SendHeartbeatTo = testServer.URL + "/hearbeat"
	return testServer, serverErr
}

func mockGenerateAlerts() *[]byte {
	currentTime := time.Now()
	testAlert := models.Alert{
		SentOn:           currentTime.AddDate(0, 0, -1).Unix(),
		DeviceID:         "Sensor1",
		AlertNumber:      22,
		AlertDescription: "sensor disconnected",
		Severity:         "medium",
		MeshID:           "MeshID_45",
		MeshNodeID:       "MeshNodeID_45",
	}
	inputData, _ := json.Marshal(testAlert)
	return &inputData
}
