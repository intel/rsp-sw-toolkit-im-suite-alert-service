/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"log"

	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/pkg/web"
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
				"alert_number": 260,
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

func TestSendResetBaselineCompletionAlertMessageWithNoAlertNumber(t *testing.T) {
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
