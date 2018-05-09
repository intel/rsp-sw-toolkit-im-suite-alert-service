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
	"bytes"
	"context"
	"context_linux_go/core"
	"context_linux_go/core/sensing"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	reporter "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics-influxdb"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/routes"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/healthcheck"
)

const (
	heartbeatUrn      = "urn:x-intel:context:retailsensingplatform:heartbeat"
	alertsUrn         = "urn:x-intel:context:retailsensingplatform:alerts"
	jsonApplication   = "application/json;charset=utf-8"
	connectionTimeout = 15

	heartbeat = "HeartBeat"
	alert     = "Alert"
)

// Notification struct
type Notification struct {
	NotificationType    string
	NotificationMessage string
	Data                interface{}
	GatewayID           string
	Endpoint            string
}

var gateway = models.GetInstanceGateway()
var notificationChan chan Notification

func init() {

}

// nolint: gocyclo
func initSensing() {
	onSensingStarted := make(core.SensingStartedChannel, 1)
	onSensingError := make(core.ErrorChannel, 1)

	sensingOptions := core.SensingOptions{
		Server:  config.AppConfig.ContextSensing,
		Publish: true,
		Secure:  config.AppConfig.SecureMode,
		SkipCertificateVerification: config.AppConfig.SkipCertVerify,
		Application:                 config.AppConfig.ServiceName,
		OnStarted:                   onSensingStarted,
		OnError:                     onSensingError,
		Retries:                     10,
		RetryInterval:               1,
	}

	sensingSdk := sensing.NewSensing()
	sensingSdk.Start(sensingOptions)

	go func(options core.SensingOptions) {
		onHeartbeat := make(core.ProviderItemChannel, 10)
		onAlert := make(core.ProviderItemChannel, 10)

		for {
			select {
			case started := <-options.OnStarted:
				if !started.Started {
					log.WithFields(log.Fields{
						"Method": "main",
						"Action": "connecting to context broker",
						"Host":   config.AppConfig.ContextSensing,
					}).Fatal("sensing has failed to start")
				}

				log.Info("Sensing has started")
				sensingSdk.AddContextTypeListener("*:*", heartbeatUrn, &onHeartbeat, &onSensingError)
				sensingSdk.AddContextTypeListener("*:*", alertsUrn, &onAlert, &onSensingError)
				log.Info("Waiting for Heartbeat, Event, and Alert data....")

			case heartbeat := <-onHeartbeat:
				jsonBytes, err := json.MarshalIndent(*heartbeat, "", "  ")
				if err != nil {
					log.Errorf("Unable to process heartbeat")
				}

				if err := processHeartbeat(&jsonBytes); err != nil {
					log.WithFields(log.Fields{
						"Method": "main",
						"Action": "process HeartBeat",
						"Error":  err.Error(),
					}).Error("error processing heartbeat data")
				}
			case alert := <-onAlert:
				var err error
				jsonBytes, err := json.MarshalIndent(*alert, "", "  ")
				if err != nil {
					log.Errorf("Unable to process alert")
				}
				if err := processAlert(&jsonBytes); err != nil {
					log.WithFields(log.Fields{
						"Method": "main",
						"Action": "process Alert",
						"Error":  err.Error(),
					}).Error("error processing alert")
				}
			case err := <-options.OnError:
				log.Fatalf("Received sensingSdk error: %v, exiting...", err)
			}
		}
	}(sensingOptions)
}

func monitorHeartbeat(watchdogSeconds int) {

	for {
		<-time.After(time.Duration(watchdogSeconds) * time.Second)
		// we only care about Gateways that are currently registered and who have missed heartbeat
		if gateway.GetRegistrationStatus() == models.Registered && time.Since(gateway.GetLastHeartbeatSeen()) > time.Duration(watchdogSeconds)*time.Second {
			if gateway.UpdateMissedHeartBeats() {
				if gateway.GetMissedHeartBeats() >= config.AppConfig.MaxMissedHeartbeats {
					// Since we have missed the maximum amount of heartbeats, set this gateway to deregistered and send alert
					if gateway.DeregisterGateway() {
						gatewayDeregistered, gatewayID := models.GatewayDeregisteredAlert(gateway.GetLastHeartbeat())
						log.Debug("Gateway Deregistered")
						notificationChan <- Notification{
							NotificationType:    alert,
							NotificationMessage: "Gateway Deregistered Alert",
							Data:                gatewayDeregistered,
							GatewayID:           gatewayID,
						}
					}
				} else {
					// send missed heartbeat alert
					missedHeartbeat, gatewayID := models.GatewayMissedHeartbeatAlert(gateway.GetLastHeartbeat())
					log.Debug("Missed heartbeat")
					notificationChan <- Notification{
						NotificationType:    alert,
						NotificationMessage: "Missed HeartBeat Alert",
						Data:                missedHeartbeat,
						GatewayID:           gatewayID,
					}
				}
			}
		}
	}
}

func updateGatewayStatus(hb models.Heartbeat) {
	lastHeartbeatSeen := time.Now()
	lastHeartbeat := hb
	missedHeartBeats := 0
	if gateway.UpdateGatewayStatus(lastHeartbeatSeen, missedHeartBeats, lastHeartbeat) {
		if gateway.GetRegistrationStatus() == models.Pending || gateway.GetRegistrationStatus() == models.Deregistered {
			if gateway.RegisterGateway() {
				gatewayRegistered, gatewayID := models.GatewayRegisteredAlert(gateway.GetLastHeartbeat())
				log.Debug("Gateway Registered")
				notificationChan <- Notification{
					NotificationType:    alert,
					NotificationMessage: "Gateway Registered Alert",
					Data:                gatewayRegistered,
					GatewayID:           gatewayID,
				}
			}

		}
	}
}

func processHeartbeat(jsonBytes *[]byte) error {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.ProcessHeartBeat.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Success", nil)
	mUnmarshalErr := metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Unmarshal-Error", nil)

	jsoned := string(*jsonBytes)
	log.Infof("Received Heartbeat:\n%s", jsoned)

	var heartbeatEvent models.HeartbeatMessage
	err := json.Unmarshal(*jsonBytes, &heartbeatEvent)
	if err != nil {
		log.Errorf("error parsing Heartbeat: %s", err)
		mUnmarshalErr.Update(1)
		return err
	}

	updateGatewayStatus(heartbeatEvent.Value)

	notificationChan <- Notification{
		NotificationMessage: "Process HeartBeat",
		NotificationType:    heartbeat,
		Data:                heartbeatEvent.Value,
		GatewayID:           heartbeatEvent.Value.DeviceID,
	}
	log.Info("Processed heartbeat")
	mSuccess.Update(1)
	return nil
}

func processAlert(jsonBytes *[]byte) error {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.ProcessAlert.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Success", nil)
	mUnmarshalErr := metrics.GetOrRegisterGauge("RFID-Alert.ProcessAlert.Unmarshal-Error", nil)

	jsoned := string(*jsonBytes)
	log.Infof("Received alert:\n%s", jsoned)

	var data map[string]interface{}
	//gatewayID required for JWT signing but should not be sent to the cloud
	var gatewayID string
	if err := json.Unmarshal(*jsonBytes, &data); err != nil {
		log.Errorf("error parsing Alert: %s", err)
		mUnmarshalErr.Update(1)
		return err
	}
	if value, ok := data["value"].(map[string]interface{}); !ok {
		return errors.New("Type assertion failed")
	} else {
		gatewayID, ok = value["gateway_id"].(string)
		if !ok {
			return errors.New("Type assertion failed")
		}
	}

	var alertEvent models.AlertMessage
	err := json.Unmarshal(*jsonBytes, &alertEvent)
	if err != nil {
		log.Errorf("error parsing Alert: %s", err)
		mUnmarshalErr.Update(1)
		return err
	}
	notificationChan <- Notification{
		NotificationMessage: "Process Alert",
		NotificationType:    alert,
		Data:                alertEvent.Value,
		GatewayID:           gatewayID,
	}

	log.Info("Processed alert")
	mSuccess.Update(1)
	return nil
}

func notifyChannel() {
	// CloudConnector URL to send both heartbeats and alerts
	cloudConnector := config.AppConfig.CloudConnectorURL + config.AppConfig.CloudConnectorEndpoint
	notificationChanSize := config.AppConfig.NotificationChanSize

	for notification := range notificationChan {
		if len(notificationChan) >= notificationChanSize-10 {
			log.WithFields(log.Fields{
				"notificationChanSize": len(notificationChan),
				"maxChannelSize":       notificationChanSize,
			}).Warn("Channel size getting full!")
		}
		generateErr := notification.generatePayload()
		if generateErr != nil {
			log.Errorf("Problem generating payload for %s, %s", notification.NotificationType, generateErr)
		} else {
			_, err := postNotification(notification.Data, cloudConnector)
			if err != nil {
				log.Errorf("Problem sending notification for %s, %s", notification.NotificationMessage, err)
			}
		}
	}
}

func (notificationData *Notification) generatePayload() error {
	var event interface{}
	var endPoint string
	if notificationData.NotificationType == heartbeat {
		event = notificationData.Data.(models.Heartbeat)
		endPoint = config.AppConfig.HeartbeatEndpoint
	} else {
		event = notificationData.Data.(models.Alert)
		endPoint = config.AppConfig.AlertEndpoint
	}

	// Get Signed JWT for message authentication
	signedJwt, err := getSignedJwt(config.AppConfig.JwtSignerURL+config.AppConfig.JwtSignerEndpoint, notificationData.GatewayID)
	if err != nil {
		return errors.Wrapf(err, "problem getting the signed jwt")
	}

	var payload models.CloudConnectorPayload
	payload.Method = "POST"
	payload.URL = config.AppConfig.AwsURLHost + config.AppConfig.AwsURLStage + endPoint
	header := http.Header{}
	header["Content-Type"] = []string{"application/json"}
	header["Authorization"] = []string{"Bearer " + signedJwt}
	payload.Header = header
	payload.IsAsync = true
	payload.Payload = event
	notificationData.Data = payload
	return nil
}

func getSignedJwt(jwtSignerURL string, gatewayID string) (string, error) {
	jwtRequest := models.JwtSignerRequest{
		Claims: map[string]string{
			"iss": gatewayID,
		},
	}

	response, err := postNotification(jwtRequest, jwtSignerURL)
	if err != nil {
		return "", errors.Wrapf(err, "unable to make JWT http request")
	}

	jwtResponse := models.JwtSignerResponse{}
	if unmarshalErr := json.Unmarshal(response, &jwtResponse); unmarshalErr != nil {
		return "", errors.Wrapf(err, "failed to Unmarshal responseData")
	}

	return jwtResponse.Token, nil
}

func postNotification(data interface{}, to string) ([]byte, error) {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.PostNotification.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Success", nil)
	mMarshalErr := metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Marshal-Error", nil)
	mNotifyErr := metrics.GetOrRegisterGauge("RFID-Alert.PostNotification.Notify-Error", nil)

	timeout := time.Duration(connectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	mData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		mMarshalErr.Update(1)
		return nil, errors.Errorf("Payload Marshalling failed  %v", err)
	}
	request, reqErr := http.NewRequest("POST", to, bytes.NewBuffer(mData))
	if reqErr != nil {
		return nil, reqErr
	}
	request.Header.Set("content-type", jsonApplication)
	response, respErr := client.Do(request)
	if respErr != nil {
		mNotifyErr.Update(1)
		return nil, respErr
	}

	if response.StatusCode != http.StatusOK {
		mNotifyErr.Update(1)
		return nil, errors.Errorf("PostNotification failed with following response code %d %v", response.StatusCode, response.Body)

	}

	var jwtResponse []byte
	if response.Body != nil {
		jwtResponse, err = ioutil.ReadAll(response.Body)
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
	return jwtResponse, nil
}

func main() {

	// Load config variables
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	isHealthyPtr := flag.Bool("isHealthy", false, "a bool, runs a healthcheck")
	flag.Parse()

	if *isHealthyPtr {
		os.Exit(healthcheck.Healthcheck(config.AppConfig.Port))
	}

	// Initialize metrics reporting
	initMetrics()

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting application...")

	// Initialize channel with set value in config
	notificationChan = make(chan Notification, config.AppConfig.NotificationChanSize)

	initSensing()
	go monitorHeartbeat(config.AppConfig.WatchdogSeconds)
	go notifyChannel()

	// Start Webserver
	router := routes.NewRouter()

	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           ":" + config.AppConfig.Port,
		Handler:        router,
		ReadTimeout:    900 * time.Second,
		WriteTimeout:   900 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// We want to report the listener is closed.
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the listener.
	go func() {
		log.Infof("%s running", config.AppConfig.ServiceName)
		log.Infof("Listener closed : %v", server.ListenAndServe())
		wg.Done()
	}()

	// Listen for an interrupt signal from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt)

	// Wait for a signal to shutdown.
	<-osSignals

	// Create a context to attempt a graceful 5 second shutdown.
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {
		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "shutdown",
			"Timeout": timeout,
			"Message": err.Error(),
		}).Error("Graceful shutdown did not complete")

		// Looks like we timed out on the graceful shutdown, Force Kill
		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method":  "main",
				"Action":  "shutdown",
				"Message": err.Error(),
			}).Error("Error killing server")
		}
	}

	// Wait for the listener to report it is closed.
	wg.Wait()
	log.WithField("Method", "main").Info("Completed.")
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10,                     //cfg.ReportingInterval,
			config.AppConfig.TelemetryEndpoint, //cfg.ReportingEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}
