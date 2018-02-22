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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
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

func initGatewayStatusCheck(watchdogMinutes int) {

	gw.RegistrationStatus = models.Pending
	gw.MissedHeartBeats = 0
	gw.LastHeartbeatSeen = defaulttime

	for {
		<-time.After(time.Duration(watchdogMinutes) * time.Minute)
		if gw.RegistrationStatus == models.Registered {
			// we only care about GWs that are currently registered
			if gw.LastHeartbeatSeen != defaulttime {
				// we have seen a HB before,
				// see if the timespan has expired
				if time.Since(gw.LastHeartbeatSeen) > time.Duration(watchdogMinutes)*time.Minute {
					gw.MissedHeartBeats = gw.MissedHeartBeats + 1
					if gw.MissedHeartBeats >= config.AppConfig.MaxMissedHeartbeats {
						// we have missed the maximum amount of heartbeats
						// set this gateway to deregistered
						gw.RegistrationStatus = models.Deregistered
						gwDeregistered := models.NewGatewayDeregisteredAlert(gw.LastHeartbeat)
						if err := postNotification(gwDeregistered, config.AppConfig.SendAlertTo); err != nil {
							log.Errorf("Problem sending GatewayRegistered Alert: %s", err)
						}
					} else {
						gwMissedHB := models.NewGatewayMissedHeartbeatAlert(gw.LastHeartbeat)
						if err := postNotification(gwMissedHB, config.AppConfig.SendAlertTo); err != nil {
							log.Errorf("Problem sending GatewayRegistered Alert: %s", err)
						}
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

	if gw.RegistrationStatus == models.Pending {
		// we just got a heart beat, so mark it as registered
		gw.RegistrationStatus = models.Registered
		gw.FirstHeartbeatSeen = gw.LastHeartbeatSeen
		gwRegistered := models.NewGatewayRegisteredAlert(gw.LastHeartbeat)

		if err := postNotification(gwRegistered, config.AppConfig.SendAlertTo); err != nil {
			log.Errorf("Problem sending GatewayRegistered Alert: %s", err)
		}
	}
}

func processHeartbeat(jsonBytes *[]byte) error {
	jsoned := string(*jsonBytes)
	log.Infof("Received Heartbeat:\n%s", jsoned)

	var hb models.HeartBeatMessage
	err := json.Unmarshal(*jsonBytes, &hb)
	if err != nil {
		fmt.Println("error parsing Heartbeat:", err)
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
	eventEndpoint := config.AppConfig.SendAlertTo
	if err := postNotification(jsoned, eventEndpoint); err != nil {
		log.Errorf("Problem sending Event: %s", err)
		return err
	}

	log.Infof("Sent alert to %s", eventEndpoint)
	return nil
}

func postNotification(data interface{}, to string) error {
	timeout := time.Duration(connectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}

	mData, err := json.MarshalIndent(data, "", "    ")
	if err == nil {
		request, _ := http.NewRequest("POST", to, bytes.NewBuffer(mData))
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

	go initGatewayStatusCheck(config.AppConfig.WatchdogMinutes)

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
