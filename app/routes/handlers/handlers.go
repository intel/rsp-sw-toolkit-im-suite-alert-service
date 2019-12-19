/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/alert"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/models"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/routes/schemas"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/pkg/web"
	metrics "github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	"github.com/pkg/errors"
)

// Alerts represents the User API method handler set.
type Alerts struct {
}

// GetIndex verifies check health
// nolint :unparam
func (alerts *Alerts) GetIndex(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, "Alert Service", http.StatusOK)
	return nil
}

// SendAlertMessageToCloudConnector post the alert message in the request JSON payload to cloud connector
func (alerts *Alerts) SendAlertMessageToCloudConnector(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	// Metrics
	metrics.GetOrRegisterGauge("Alerts.SendAlertMessageToCloudConnector.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("Alerts.SendAlertMessageToCloudConnector.Latency", nil).Update(time.Since(startTime))

	mSendAlertLatency := metrics.GetOrRegisterTimer("Alerts.SendAlertMessageToCloudConnector.SendAlert-Latency", nil)

	mSuccess := metrics.GetOrRegisterGauge("Alerts.SendAlertMessageToCloudConnector.Success", nil)
	mSendCloudConnectorErr := metrics.GetOrRegisterGauge("Alerts.SendAlertMessageToCloudConnector.Send-Error", nil)
	mProcessRequestErr := metrics.GetOrRegisterGauge("Alerts.SendAlertMessageToCloudConnector.ProcessRequest-Error", nil)
	mInputValErr := metrics.GetOrRegisterGauge("Alerts.SendAlertMessageToCloudConnector.Input-Validation-Error", nil)

	var payload models.AlertMessage
	inputValErrs, err := readAndValidateRequest(request, schemas.AlertMessageSchema, &payload)
	if err != nil {
		mProcessRequestErr.Update(1)
		return err
	}
	if inputValErrs != nil {
		mInputValErr.Update(1)
		web.Respond(ctx, writer, inputValErrs, http.StatusBadRequest)
		return errors.New("could not validate request alertmessage schema")
	}

	sentCloudConnectorTimer := time.Now()
	populateAlertNotificationPayload(&payload)
	alertBytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		web.Respond(ctx, writer, inputValErrs, http.StatusBadRequest)
		return errors.New("could not marshal the payload json bytes")
	}

	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	processAlertErr := alert.ProcessAlert(&alertBytes, notificationChan)

	if processAlertErr != nil {
		mSendCloudConnectorErr.Update(1)
		web.Respond(ctx, writer, inputValErrs, http.StatusInternalServerError)
		return errors.New("process alert error")
	}
	go alert.NotifyChannel(notificationChan)

	mSendAlertLatency.Update(time.Since(sentCloudConnectorTimer))
	mSuccess.Update(1)

	responseData := "Alert Service has successfully process alertMessage to cloud connector"
	web.Respond(ctx, writer, responseData, http.StatusOK)
	return nil
}

// nolint: unparam
func readAndValidateRequest(request *http.Request, schema string, v interface{}) (interface{}, error) {
	// Reading request
	body := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, body)
	if err != nil {
		return nil, errors.Wrap(web.ErrValidation, err.Error())
	}

	// Unmarshal request as json
	if err = json.Unmarshal(body, &v); err != nil {
		return nil, errors.Wrap(web.ErrValidation, err.Error())
	}

	// Validate json against schema
	schemaValidatorResult, err := schemas.ValidateSchemaRequest(body, schema)
	if err != nil {
		return nil, err
	}
	if !schemaValidatorResult.Valid() {
		result := schemas.BuildErrorsString(schemaValidatorResult.Errors())
		return result, nil
	}

	return nil, nil
}

func populateAlertNotificationPayload(alertPayload *models.AlertMessage) {
	// use system time
	alertPayload.Datetime = time.Now()
	// override 0
	if alertPayload.Value.AlertNumber == 0 {
		alertPayload.Value.AlertNumber = 686
	}
}
