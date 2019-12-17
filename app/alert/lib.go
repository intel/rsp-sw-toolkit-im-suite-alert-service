/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package alert

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/models"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// AlertType is type for alert
	AlertType = "Alert"
	//
	jsonApplication = "application/json;charset=utf-8"
	//
	connectionTimeout = 15
	// Not Whitelisted Alert Type
	NotWhitelisted = 401
)

// ProcessAlert takes alert json bytes and post to notification channel
func ProcessAlert(jsonBytes *[]byte, notificationChan chan Notification) error {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.ProcessAlert.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Success", nil)
	mUnmarshalErr := metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Unmarshal-Error", nil)

	jsoned := string(*jsonBytes)
	log.Debugf("Received alert:\n%s", jsoned)

	var data map[string]interface{}

	var gatewayID string
	if err := json.Unmarshal(*jsonBytes, &data); err != nil {
		log.Errorf("error parsing Alert %s", err)
		mUnmarshalErr.Update(1)
		return err
	}

	gatewayID, ok := data["gateway_id"].(string)
	if !ok {
		// ASN Alert will not contain gateway id
		log.Warn("This may not be an issue, but received Alert without gateway id.")
	}

	var alertEvent models.Alert
	err := json.Unmarshal(*jsonBytes, &alertEvent)
	if err != nil {
		log.Errorf("error parsing Alert %s", err)
		mUnmarshalErr.Update(1)
		return err
	}
	go func() {
		notificationChan <- Notification{
			NotificationMessage: "Process Alert",
			NotificationType:    AlertType,
			Data:                alertEvent,
			GatewayID:           gatewayID,
			Endpoint:            config.AppConfig.AlertDestination,
		}
	}()

	log.Debug("Processed alert")
	mSuccess.Update(1)
	return nil
}

// NotifyChannel iterates through messages in the notification channel and post to cloud connector
func NotifyChannel(notificationChan chan Notification) {
	// CloudConnector URL to send alerts
	cloudConnectorEndpoint := config.AppConfig.CloudConnectorURL + config.AppConfig.CloudConnectorEndpoint
	notificationChanSize := config.AppConfig.NotificationChanSize

	for notification := range notificationChan {
		if len(notificationChan) >= notificationChanSize-10 {
			log.WithFields(log.Fields{
				"notificationChanSize": len(notificationChan),
				"maxChannelSize":       notificationChanSize,
			}).Warn("Channel size getting full!")
		}
		generateErr := notification.GeneratePayload()
		if generateErr != nil {
			log.Errorf("Problem generating payload for %s, %s", notification.NotificationType, generateErr)
		} else {

			dataBytes, err := json.MarshalIndent(notification.Data, "", "    ")
			if err != nil {
				log.Errorf("unable to marshal. %s", err)
			}

			cloudConnectorPayload := getCloudConnectorPayload(dataBytes)
			if cloudConnectorPayload.URL != "" {
				cloudConnectorPayloadBytes, err := json.MarshalIndent(cloudConnectorPayload, "", "    ")
				if err != nil {
					log.Errorf("unable to marshal. %s", err)
				}
				_, err = PostNotification(cloudConnectorPayloadBytes, cloudConnectorEndpoint)
				if err != nil {
					log.Errorf("Problem sending notification for %s, %s", notification.NotificationMessage, err)
				}
			} else {
				log.Warn("Payload for Cloud Connector doesn't include a destination URL.  Not sending POST message to Cloud Connector.")
			}
		}
	}
}

// PostNotification post notification data vial http call to the toURL
func PostNotification(data []byte, toURL string) ([]byte, error) {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.PostNotification.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Success", nil)
	mNotifyErr := metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Notify-Error", nil)

	timeout := time.Duration(connectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	log.Debugf("Payload to cloud-connector after marshalling:\n%s", string(data))
	request, err := http.NewRequest("POST", toURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("content-type", jsonApplication)
	response, respErr := client.Do(request)
	if respErr != nil {
		mNotifyErr.Update(1)
		return nil, respErr
	}

	if response.StatusCode != http.StatusOK {
		mNotifyErr.Update(1)
		return nil, errors.Errorf("PostNotification failed with following response code %d", response.StatusCode)

	}

	var responseData []byte
	if response.Body != nil {
		responseData, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to ReadALL response.Body")
		}
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method": "postNotification",
				"Action": "response.Body.Close()",
			}).Info(err.Error())
		}
	}()

	log.Debug("Notification posted")
	mSuccess.Update(1)
	return responseData, nil
}

func getCloudConnectorPayload(dataBytes []byte) models.CloudConnectorPayload {

	var cloudConnectorPayload models.CloudConnectorPayload

	err := json.Unmarshal(dataBytes, &cloudConnectorPayload)
	if err != nil {
		log.Errorf("unable to unmarshal. %s", err)
		return models.CloudConnectorPayload{}
	}

	// Unmarshal the auth data into the auth model for the cloud connector service to consume.
	if config.AppConfig.AlertDestinationAuthEndpoint != "" &&
		config.AppConfig.AlertDestinationAuthType != "" &&
		config.AppConfig.AlertDestinationClientID != "" &&
		config.AppConfig.AlertDestinationClientSecret != "" {
		// Encode the endpoint credentials as base64
		authDataString := config.AppConfig.AlertDestinationClientID + ":" + config.AppConfig.AlertDestinationClientSecret
		authData := "basic " + base64.StdEncoding.EncodeToString([]byte(authDataString))

		var newAuth models.Auth
		newAuth.Endpoint = config.AppConfig.AlertDestinationAuthEndpoint
		newAuth.AuthType = config.AppConfig.AlertDestinationAuthType
		newAuth.Data = authData

		cloudConnectorPayload.Auth = newAuth
	}

	return cloudConnectorPayload
}
