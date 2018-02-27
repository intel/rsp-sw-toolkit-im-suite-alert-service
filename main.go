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
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/routes"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/healthcheck"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/web"
	"github.impcloud.net/Responsive-Retail-Inventory/utilities/gojsonschema"
)

const (
	heartbeatUrn      = "urn:x-intel:context:retailsensingplatform:heartbeat"
	alertsUrn         = "urn:x-intel:context:retailsensingplatform:alerts"
	jsonApplication   = "application/json;charset=utf-8"
	connectionTimeout = 15
)

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
		onHeartbeat := make(core.ProviderItemChannel)
		onAlert := make(core.ProviderItemChannel)

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

var gw models.GatewayStatus
var defaulttime = time.Time{}

func initGatewayStatusCheck(watchdogSeconds int) {

	gw.RegistrationStatus = models.Pending
	gw.MissedHeartBeats = 0
	gw.LastHeartbeatSeen = defaulttime

	for {
		<-time.After(time.Duration(watchdogSeconds) * time.Second)
		if gw.RegistrationStatus == models.Registered {
			// we only care about GWs that are currently registered
			if gw.LastHeartbeatSeen != defaulttime {
				// we have seen a HB before,
				// see if the timespan has expired
				if time.Since(gw.LastHeartbeatSeen) > time.Duration(watchdogSeconds)*time.Second {
					gw.MissedHeartBeats = gw.MissedHeartBeats + 1
					if gw.MissedHeartBeats >= config.AppConfig.MaxMissedHeartbeats {
						// Since we have missed the maximum amount of heartbeats, set this gateway to deregistered and send alert
						gw.RegistrationStatus = models.Deregistered
						gwDeregistered := models.NewGatewayDeregisteredAlert(gw.LastHeartbeat)
						if err := postNotification(gwDeregistered, config.AppConfig.SendAlertTo); err != nil {
							log.Errorf("Problem sending GatewayDeregistered Alert: %s", err)
						}
						log.Debug("Gateway Deregistered")
					} else {
						gwMissedHB := models.NewGatewayMissedHeartbeatAlert(gw.LastHeartbeat)
						if err := postNotification(gwMissedHB, config.AppConfig.SendAlertTo); err != nil {
							log.Errorf("Problem sending Missed heartbeat Alert: %s", err)
						}
						log.Debug("Missed heartbeat")
					}
				}
			}
		} // else this GW is not registered, so ignore it
	}
}

func updateGatewayStatus(hb models.HeartBeatMessage) {
	// we got a heartbeat, update the last seen
	gw.LastHeartbeatSeen = time.Now()
	gw.LastHeartbeat = hb
	// since we just got a heartbeat, there are no missed hb
	gw.MissedHeartBeats = 0

	if gw.RegistrationStatus == models.Pending || gw.RegistrationStatus == models.Deregistered {
		// we just got a heart beat, so mark it as registered
		gw.RegistrationStatus = models.Registered
		gw.FirstHeartbeatSeen = gw.LastHeartbeatSeen
		gwRegistered := models.NewGatewayRegisteredAlert(gw.LastHeartbeat)
		log.Debug("Gateway Registered")
		if err := postNotification(gwRegistered, config.AppConfig.SendAlertTo); err != nil {
			log.Errorf("Problem sending GatewayRegistered Alert: %s", err)
		}
	}
}

func processHeartbeat(jsonBytes *[]byte) error {
	jsoned := string(*jsonBytes)
	log.Infof("Received Heartbeat:\n%s", jsoned)

	validationErr := ValidateSchemaRequest(*jsonBytes, models.HeartBeatSchema)
	if validationErr != nil {
		return validationErr
	}

	var hb models.HeartBeatMessage
	err := json.Unmarshal(*jsonBytes, &hb)
	if err != nil {
		log.Errorf("error parsing Heartbeat: %s", err)
		return err
	}

	updateGatewayStatus(hb)
	heartbeatEndpoint := config.AppConfig.SendHeartbeatTo
	if err := postNotification(jsoned, heartbeatEndpoint); err != nil {
		log.Errorf("Problem sending Heartbeat: %s", err)
		return err
	}

	log.Infof("Sent Heartbeat to %s", heartbeatEndpoint)
	return nil
}

func processAlert(jsonBytes *[]byte) error {
	jsoned := string(*jsonBytes)
	log.Infof("Received alert:\n%s", jsoned)

	validationErr := ValidateSchemaRequest(*jsonBytes, models.AlertSchema)
	if validationErr != nil {
		return validationErr
	}

	eventEndpoint := config.AppConfig.SendAlertTo
	if err := postNotification(jsoned, eventEndpoint); err != nil {
		log.Errorf("Problem sending Event: %s", err)
		return err
	}

	log.Infof("Sent alert to %s", eventEndpoint)
	return nil
}

//ErrReport is used to wrap schema validation errors int json object
type ErrReport struct {
	Field       string      `json:"field"`
	ErrorType   string      `json:"errortype"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// ValidateSchemaRequest validates the api request body with the required json schema
func ValidateSchemaRequest(jsonBody []byte, schema string) error {
	if len(jsonBody) == 0 {
		return errors.Wrapf(web.ErrInvalidInput, "request body cannot be empty")
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewBytesLoader(jsonBody)

	validatorResult, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return errors.Wrapf(web.ErrInvalidInput, err.Error())
	}
	if !validatorResult.Valid() {
		var error ErrReport
		var errorSlice []ErrReport
		for _, err := range validatorResult.Errors() {
			// ignore extraneous "number_one_of" error
			if err.Type() == "number_one_of" {
				continue
			}
			error.Description = err.Description()
			error.Field = err.Field()
			error.ErrorType = err.Type()
			error.Value = err.Value()
			errorSlice = append(errorSlice, error)
		}
		log.Errorf("Validaton of schema failed %s", errorSlice)
		return errors.Wrapf(web.ErrInvalidInput, "Validation of schema failed %s", errorSlice)
	}
	return nil
}

func postNotification(data interface{}, to string) error {
	timeout := time.Duration(connectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	mData, err := json.MarshalIndent(data, "", "    ")
	if err == nil {
		request, reqErr := http.NewRequest("POST", to, bytes.NewBuffer(mData))
		if reqErr != nil {
			return reqErr
		}
		request.Header.Set("content-type", jsonApplication)
		response, err := client.Do(request)
		if err != nil ||
			response.StatusCode != http.StatusOK {
			return err
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				log.WithFields(log.Fields{
					"Method": "postNotification",
					"Action": "post notification to notification service",
				}).Info(err.Error())
			}
		}()
	}
	log.Debug("Notification posted")
	return nil
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

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting application...")

	initSensing()

	go initGatewayStatusCheck(config.AppConfig.WatchdogSeconds)

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
