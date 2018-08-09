/*
 * INTEL CONFIDENTIAL
 * Copyright (2016, 2017) Intel Corporation.
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
package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"log"

	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/web"
)

func TestGetIndex(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	alerts := Alerts{}
	handler := web.Handler(alerts.GetIndex)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected 200 response")
	}
	log.Print(recorder.Body.String())
	if recorder.Body.String() != "\"RFID Alert Service\"" {
		t.Fatalf("Expected body to equal RFID Alert Service")
	}
}

func TestSendResetBaselineCompletionAlertMessageOk(t *testing.T) {
	resetBaselineAlertMessage := []byte(`{
			"application": "test app",
			"value": {
				"sent_on": 1531522680000,
				"alert_description": "reset baseline test alert message",
				"severity": "warning",
				"optional": "tag mongo db collection with 1000 tags has been deleted"
			}
		}`)
	request, err := http.NewRequest(http.MethodPost, "/alertmessage", bytes.NewBuffer(resetBaselineAlertMessage))
	if err != nil {
		t.Fatalf("Unable to create a new HTTP request: %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	alert := Alerts{}
	handler := web.Handler(alert.SendAlertMessageToCloudConnector)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Success expected: %d Actual: %d", http.StatusOK, recorder.Code)
	}
}

func TestSendAlertMessageBadInputs(t *testing.T) {
	missingFieldInput := []byte(`{
			"application": "test app",
			"value": {
				"sent_on": 1531522680000
			}
		}`)
	request, err := http.NewRequest(http.MethodPost, "/alertmessage", bytes.NewBuffer(missingFieldInput))
	if err != nil {
		t.Fatalf("Unable to create a new HTTP request: %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	alert := Alerts{}
	handler := web.Handler(alert.SendAlertMessageToCloudConnector)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Bad Request for invalid json input expected: %d Actual: %d", http.StatusBadRequest, recorder.Code)
	}

	corruptedInput := []byte(`{
		"xxxx": "corrupted",
		"xxx": {
			"sent_on": 8jcdsdf
		}
	}`)
	request, err = http.NewRequest(http.MethodPost, "/alertmessage", bytes.NewBuffer(corruptedInput))
	if err != nil {
		t.Fatalf("Unable to create a new HTTP request: %s", err.Error())
	}

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Bad Request for corrupted json input expected: %d Actual: %d", http.StatusBadRequest, recorder.Code)
	}
}
