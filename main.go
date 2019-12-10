/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	golog "log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	edgexModels "github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/asn"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/models"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/routes"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/pkg/utils"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"
	reporter "github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics-influxdb"
)

var gateway = models.GetInstanceGateway()

const (
	serviceKey = "rfid-alert-service"

	// Reading names
	heartbeat   = "controller_heartbeat" // note: the current mqtt-device-service does not forward these
	deviceAlert = "device_alert"
	asnData     = "ASN_data"
)

// only Readings with these names are received.
var readingFilter = []string{heartbeat, deviceAlert, asnData}

type reading struct {
	Topic  string                 `json:"topic"`
	Params map[string]interface{} `json:"params"`
}

// SkuMapping struct for the SkuMapping
type SkuMapping struct {
	url string
}

type notificationChannel struct {
	channel chan alert.Notification
}

// NewSkuMapping initialize new SkuMapping
func NewSkuMapping(url string) *SkuMapping {
	return &SkuMapping{
		url: url,
	}
}

func monitorHeartbeat(watchdogSeconds int, notificationChan chan alert.Notification) {

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
						go func() {
							notificationChan <- alert.Notification{
								NotificationType:    alert.AlertType,
								NotificationMessage: "Gateway Deregistered Alert",
								Data:                gatewayDeregistered,
								GatewayID:           gatewayID,
								Endpoint:            config.AppConfig.AlertDestination,
							}
						}()
					}
				} else {
					// send missed heartbeat alert
					missedHeartbeat, gatewayID := models.GatewayMissedHeartbeatAlert(gateway.GetLastHeartbeat())
					log.Debug("Missed heartbeat")
					go func() {
						notificationChan <- alert.Notification{
							NotificationType:    alert.AlertType,
							NotificationMessage: "Missed HeartBeat Alert",
							Data:                missedHeartbeat,
							GatewayID:           gatewayID,
							Endpoint:            config.AppConfig.AlertDestination,
						}
					}()
				}
			}
		}
	}
}

func updateGatewayStatus(hb models.Heartbeat, notificationChan chan alert.Notification) {
	lastHeartbeatSeen := time.Now()
	lastHeartbeat := hb
	missedHeartBeats := 0
	if gateway.UpdateGatewayStatus(lastHeartbeatSeen, missedHeartBeats, lastHeartbeat) {
		if gateway.GetRegistrationStatus() == models.Pending || gateway.GetRegistrationStatus() == models.Deregistered {
			if gateway.RegisterGateway() {
				gatewayRegistered, gatewayID := models.GatewayRegisteredAlert(gateway.GetLastHeartbeat())
				log.Debug("Gateway Registered")
				go func() {
					notificationChan <- alert.Notification{
						NotificationType:    alert.AlertType,
						NotificationMessage: "Gateway Registered Alert",
						Data:                gatewayRegistered,
						GatewayID:           gatewayID,
						Endpoint:            config.AppConfig.AlertDestination,
					}
				}()
			}

		}
	}
}

func processHeartbeat(jsonBytes *[]byte, notificationChan chan alert.Notification) error {
	// Metrics
	metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Attempt", nil).Update(1)
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("RFID-Alert.ProcessHeartBeat.Latency", nil).UpdateSince(startTime)
	mSuccess := metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Success", nil)
	mUnmarshalErr := metrics.GetOrRegisterGauge("RFID-Alert.ProcessHeartBeat.Unmarshal-Error", nil)

	jsoned := string(*jsonBytes)
	log.Debugf("Received Heartbeat:\n%s", jsoned)

	var heartbeatEvent models.Heartbeat
	err := json.Unmarshal(*jsonBytes, &heartbeatEvent)
	if err != nil {
		log.Errorf("error parsing Heartbeat %s", err)
		mUnmarshalErr.Update(1)
		return err
	}

	updateGatewayStatus(heartbeatEvent, notificationChan)

	// Forward the heartbeat to the notification channel
	go func() {
		notificationChan <- alert.Notification{
			NotificationMessage: "Process Heartbeat",
			NotificationType:    models.HeartbeatType,
			Data:                heartbeatEvent,
			GatewayID:           heartbeatEvent.DeviceID,
			Endpoint:            config.AppConfig.HeartbeatDestination,
		}
	}()

	log.Debug("Processed heartbeat")
	mSuccess.Update(1)
	return nil
}

func (skuMapping SkuMapping) processShippingNotice(jsonBytes *[]byte, notificationChan chan alert.Notification) error {
	mRRSAsnsNotWhitelisted := metrics.GetOrRegisterGaugeCollection("Rfid-Alert.ASNsNotWhitelisted", nil)
	log.Debugf("Received advanced shipping notice data:\n%s", string(*jsonBytes))

	var advanceShippingNotices []models.AdvanceShippingNotice
	err := json.Unmarshal((*jsonBytes), &advanceShippingNotices)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal data")
	}

	productIDs, err := extractProductIDs(advanceShippingNotices)

	if len(productIDs) == 0 {
		log.Debug("Received zero productIDs in shipping notice.")
		return nil
	}
	oDataQuery := buildODataQuery(productIDs)
	if err != nil {
		log.Errorf("Problem converting shipping notice data to GTINs or Proprietary IDs: %s", err)
	}

	var whitelistedProductIDs []string
	var stringBytes bytes.Buffer
	if len(oDataQuery) > config.AppConfig.BatchSizeMax {
		var batchSize = config.AppConfig.BatchSizeMax
		var start = 0
		for start < len(oDataQuery) {
			stringBytes.WriteString(strings.Join(oDataQuery[start:start+batchSize], " or "))
			productsFromSkuMapping, callErr := MakeGetCallToSkuMapping(stringBytes.String(), skuMapping.url)
			if callErr != nil {
				log.WithFields(log.Fields{
					"Method": "processShippingNotice",
					"Action": "Calling MakeGetCallToSkuMapping",
					"Error":  callErr.Error(),
				}).Error(callErr)
				return errors.Wrapf(callErr, "unable to get list of productIDs from mapping sku service")
			}
			whitelistedProductIDs = append(whitelistedProductIDs, productsFromSkuMapping...)
			start += batchSize
		}
	} else {
		stringBytes.WriteString(strings.Join(oDataQuery, " or "))
		productsFromSkuMapping, callErr := MakeGetCallToSkuMapping(stringBytes.String(), skuMapping.url)
		if callErr != nil {
			log.WithFields(log.Fields{
				"Method": "processShippingNotice",
				"Action": "Calling MakeGetCallToSkuMapping",
				"Error":  callErr.Error(),
			}).Error(callErr)
			return errors.Wrapf(callErr, "unable to get list of productIDs from mapping sku service")
		}
		whitelistedProductIDs = append(whitelistedProductIDs, productsFromSkuMapping...)
	}

	notWhitelisted := utils.Filter(productIDs, func(v string) bool {
		return !utils.Include(whitelistedProductIDs, v)
	})

	notWhitelisted = utils.RemoveDuplicates(notWhitelisted)

	if len(notWhitelisted) > 0 {
		asnList, err := models.ConvertToASNList(notWhitelisted)
		if err != nil {
			log.WithFields(log.Fields{
				"Method": "processShippingNotice",
				"Action": "Calling ConvertToASNList",
				"Error":  err.Error(),
			}).Error(err)
			return errors.Wrapf(err, "unable to convert to asns")
		}

		mRRSAsnsNotWhitelisted.Add(int64(len(notWhitelisted)))

		alertBytes, err := asn.GenerateNotWhitelistedAlert(asnList)
		if err != nil {
			log.WithFields(log.Fields{
				"Method": "processShippingNotice",
				"Action": "Calling GenerateNotWhitelistedAlert",
				"Error":  err.Error(),
			}).Error(err)
			return errors.Wrapf(err, "unable to generate alert to send for asns not whitelisted")
		}

		log.Errorf("Received asn with tags not whitelisted. %s", notWhitelisted)
		if config.AppConfig.SendNotWhitelistedAlert {
			if processErr := alert.ProcessAlert(&alertBytes, notificationChan); processErr != nil {
				log.WithFields(log.Fields{
					"Method": "processShippingNotice",
					"Action": "Calling ProcessAlert",
					"Error":  err.Error(),
				}).Error(err)
				return errors.Wrapf(err, "unable to process alert for shipping notice")
			}
		}
	}

	return nil
}

func extractProductIDs(advanceShippingNotices []models.AdvanceShippingNotice) ([]string, error) {
	var productIDs []string
	for _, advanceShippingNotice := range advanceShippingNotices {
		for _, item := range advanceShippingNotice.Items {
			productIDs = append(productIDs, item.ProductID)
		}
	}
	return productIDs, nil
}

func buildODataQuery(productIDs []string) []string {
	var queries []string
	for _, productID := range productIDs {
		queries = append(queries, "(productList.productId eq '"+productID+"')")
	}
	return queries
}

// MakeGetCallToSkuMapping makes call to the Sku Mapping service to retrieve list of products
func MakeGetCallToSkuMapping(stringBytes string, skuUrl string) ([]string, error) {
	// Metrics
	metrics.GetOrRegisterMeter(`RfidAlertService.MakeGetCallToSkuMapping.Attempt`, nil).Mark(1)
	mSuccess := metrics.GetOrRegisterGauge(`RfidAlertService.MakeGetCallToSkuMapping.Success`, nil)
	mGetErr := metrics.GetOrRegisterGauge(`RfidAlertService.MakeGetCallToSkuMapping.makePostCall-Error`, nil)
	mStatusErr := metrics.GetOrRegisterGauge(`RfidAlertService.MakeGetCallToSkuMapping.requestStatusCode-Error`, nil)
	mGetLatency := metrics.GetOrRegisterTimer(`RfidAlertService.MakeGetCallToSkuMapping.makePostCall-Latency`, nil)

	timeout := time.Duration(15) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	urlEncode := &url.URL{Path: stringBytes}
	urlString := skuUrl + "?$filter=" + urlEncode.String() + "&$select=productList.productId"

	log.Debugf("Call mapping service endpoint: %s", urlString)

	request, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Make New HTTP GET request",
			"Error":  err.Error(),
		}).Error(err)
		return nil, errors.Wrapf(err, "unable to create a new GET request")
	}

	getTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		mGetErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Make HTTP GET request",
			"Error":  err.Error(),
		}).Error(err)
		return nil, errors.Wrapf(err, "unable to get description from mapping service")
	}
	defer func() {
		if respErr := response.Body.Close(); respErr != nil {
			log.WithFields(log.Fields{
				"Method": "makeGetCall",
			}).Warning("Failed to close response.")
		}
	}()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read the response body")
	}

	if response.StatusCode != http.StatusOK {
		mStatusErr.Update(1)
		log.WithFields(log.Fields{
			"Method": "MakeGetCallToSkuMapping",
			"Action": "Response code: " + strconv.Itoa(response.StatusCode),
			"Error":  fmt.Errorf("Response code: %d", response.StatusCode),
		}).Error(err)
		return nil, errors.Wrapf(errors.New("execution error"), "StatusCode %d , Response %s",
			response.StatusCode, string(responseData))
	}
	mGetLatency.UpdateSince(getTimer)
	mSuccess.Update(1)

	var result models.SkuMappingResponse
	if unmarshalErr := json.Unmarshal(responseData, &result); unmarshalErr != nil {
		return nil, errors.New("failed to Unmarshal responsedata")
	}

	var productIDs []string

	for _, productData := range result.ProdData {
		for _, productList := range productData.ProductList {
			productIDs = append(productIDs, productList.ProductID)
		}
	}

	return productIDs, nil
}

func main() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Load config variables
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	// Initialize metrics reporting
	initMetrics()

	setLoggingLevel(config.AppConfig.LoggingLevel)

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting application...")

	// Initialize channel with set value in config
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	receiveZmqEvents(notificationChan)
	go monitorHeartbeat(config.AppConfig.WatchdogSeconds, notificationChan)
	go alert.NotifyChannel(notificationChan)

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

func errorHandler(message string, err error, errorGauge *metrics.Gauge) {
	if err != nil {
		(*errorGauge).Update(1)
		log.WithFields(log.Fields{
			"Method": "main",
			"Error":  err.Error(),
		}).Error(message)
	}
}

func receiveZmqEvents(notificationChan chan alert.Notification) {

	chann := notificationChannel{channel: notificationChan}

	go func() {

		//Initialized EdgeX apps functionSDK
		edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
		if err := edgexSdk.Initialize(); err != nil {
			edgexSdk.LoggingClient.Error(fmt.Sprintf("SDK initialization failed: %v\n", err))
			os.Exit(-1)
		}

		err := edgexSdk.SetFunctionsPipeline(
			edgexSdk.ValueDescriptorFilter(readingFilter),
			chann.processEvents,
		)
		if err != nil {
			edgexSdk.LoggingClient.Error("error: ", err.Error())
			os.Exit(-1)
		}

		err = edgexSdk.MakeItRun()
		if err != nil {
			edgexSdk.LoggingClient.Error("MakeItRun returned error: ", err.Error())
			os.Exit(-1)
		}

	}()
}

func (chann notificationChannel) processEvents(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {

	mRRSProcessShippingNoticeError := metrics.GetOrRegisterGauge("Rfid-Alert.ProcessShippingNoticeError", nil)

	if len(params) < 1 {
		return false, nil
	}

	event := params[0].(edgexModels.Event)

	if len(event.Readings) < 1 {
		return false, nil
	}

	// note: only readings with names matching the readingFilter are received
	switch event.Readings[0].Name {
	case asnData:
		logrus.Debugf(fmt.Sprintf("ASN data received: %s", event))
		data, err := base64.StdEncoding.DecodeString(event.Readings[0].Value)
		if err != nil {
			errorHandler("error decoding shipping notice data", err, &mRRSProcessShippingNoticeError)
			return false, nil
		}
		skuMapping := NewSkuMapping(config.AppConfig.MappingSkuURL + config.AppConfig.MappingSkuEndpoint)
		if err := skuMapping.processShippingNotice(&data, chann.channel); err != nil {
			errorHandler("error processing shipping notice data", err, &mRRSProcessShippingNoticeError)
			return false, nil
		}

		return false, nil
	case heartbeat:
		parsedReading, err := parseReadingValue(&event.Readings[0])
		if err != nil {
			log.WithFields(log.Fields{"Method": "parseReadingValue"}).Error(err.Error())
			return false, nil
		}
		jsonBytes, err := json.MarshalIndent(&parsedReading.Params, "", "  ")
		if err != nil {
			log.Errorf("Unable to process heartbeat. Error: %s", err.Error())
			return false, nil
		}

		if err := processHeartbeat(&jsonBytes, chann.channel); err != nil {
			log.WithFields(log.Fields{
				"Method": "main",
				"Action": "process HeartBeat",
				"Error":  err.Error(),
			}).Error("error processing heartbeat data")
			return false, nil
		}
	case deviceAlert:
		parsedReading, err := parseReadingValue(&event.Readings[0])
		if err != nil {
			log.WithFields(log.Fields{"Method": "parseReadingValue"}).Error(err.Error())
			return false, nil
		}
		jsonBytes, err := json.MarshalIndent(&parsedReading.Params, "", "  ")
		if err != nil {
			log.Errorf("Unable to process alert")
			return false, nil
		}
		if err := alert.ProcessAlert(&jsonBytes, chann.channel); err != nil {
			log.WithFields(log.Fields{
				"Method": "main",
				"Action": "process Alert",
				"Error":  err.Error(),
			}).Error("error processing alert")
			return false, nil
		}
	}

	return false, nil
}

func parseReadingValue(read *edgexModels.Reading) (*reading, error) {

	readingObj := reading{}

	if err := json.Unmarshal([]byte(read.Value), &readingObj); err != nil {
		return nil, err
	}

	return &readingObj, nil

}

func setLoggingLevel(loggingLevel string) {
	switch strings.ToLower(loggingLevel) {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}
