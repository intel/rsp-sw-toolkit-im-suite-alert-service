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

	edgex "github.com/edgexfoundry/go-mod-core-contracts/models"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/models"
)

func TestMain(m *testing.M) {
	if err := config.InitConfig(); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	os.Exit(m.Run())
}

func TestProcessHeartbeat(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	inputData := mockGenerateHeartbeat()
	heartBeatError := processHeartbeat(&inputData, notificationChan)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}
	go alert.NotifyChannel(notificationChan)
}

func TestGatewayStatus(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	watchdogSeconds := 1
	//Starting gateway status check in separate goroutine
	go monitorHeartbeat(watchdogSeconds, notificationChan)
	missedHeartBeats := config.AppConfig.MaxMissedHeartbeats

	// check for gateway registered alert
	inputData := mockGenerateHeartbeat()
	heartBeatError := processHeartbeat(&inputData, notificationChan)
	if heartBeatError != nil {
		t.Errorf("Error processing heartbeat %s", heartBeatError)
	}
	if gateway.RegistrationStatus != models.Registered {
		t.Error("Failed to register gateway")
	}

	//Delay heartbeat by 3 seconds to check the functionality of missed heartbeat and gateway deregistered alert
	for i := 0; i <= missedHeartBeats; i++ {
		heartBeatError := processHeartbeat(&inputData, notificationChan)
		if heartBeatError != nil {
			t.Errorf("Error processing heartbeat %s", heartBeatError)
		}
		time.Sleep(2 * time.Second)
	}
	// looping through notification channel to make sure we get Gateway deregistered alert before checking for error conditions
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

func TestHeartbeatAlert(t *testing.T) {
	input := mockGenerateHeartbeat()
	heartbeat, err := generateHeartbeatModel(input)
	if err != nil {
		t.Fatalf("Error generating heartbeat %s", err)
	}
	heartbeatAlert, deviceID := models.GatewayRegisteredAlert(heartbeat)
	if deviceID != heartbeat.DeviceID {
		t.Error("Alert device id does not match hearbeat device id")
	}
	if heartbeatAlert.GatewayId != heartbeat.DeviceID {
		t.Error("Alert gateway device id does not match hearbeat device id")
	}
	if len(heartbeatAlert.Facilities) != len(heartbeat.Facilities) {
		t.Error("Number of alert facilitites does not match heartbeat facilities")
	}

	var heartbeatFacilities []string
	heartbeatFacilities = append(heartbeatFacilities, heartbeat.Facilities...)

	if !reflect.DeepEqual(heartbeatAlert.Facilities, heartbeatFacilities) {
		t.Error("Facilities from alert is not the same as heartbeat facilities")
	}

	input = mockGenerateHeartbeatNoFacility()
	heartbeat, err = generateHeartbeatModel(input)
	if err != nil {
		t.Fatalf("Error generating heartbeat with no facility %s", err)
	}
	heartbeatAlert, _ = models.GatewayRegisteredAlert(heartbeat)
	// Alert generated from heartbeat with no facilities should have facilities field with value "UNDEFINED_FACILITY"
	if len(heartbeatAlert.Facilities) != 1 {
		t.Errorf("Alert generated from heartbeat with no facilities should have a length of one")
	}
	if heartbeatAlert.Facilities[0] != models.UndefinedFacility {
		t.Errorf("Alert generated from heartbeat with no facilities has the wrong facility defined")
	}
}

func TestProcessShippingNoticeWRINs(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to '/skus', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
			jsonData, _ = json.Marshal(result)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")

	config.AppConfig.EpcToWrin = true
	inputData := mockGenerateShippingNoticeProprietaryIDs()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessEmptyShippingNoticeWRINs(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Error("No requests were expected because Empty WRINs")
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")

	config.AppConfig.EpcToWrin = true
	inputData := mockGenerateEmptyShippingNoticeWRINs()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessShippingNoticeGTINs(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to '/skus', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
			jsonData, _ = json.Marshal(result)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	inputData := mockGenerateShippingNoticeGTINs()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessEmptyShippingNoticeGTINs(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Error("No requests were expected because Empty WRINs")
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	inputData := mockGenerateEmptyShippingNoticeGTINs()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessShippingNoticeMixedProducts(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to '/skus', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.0, 0.0, 0.0, 0.0, "614141007349")
			jsonData, _ = json.Marshal(result)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	inputData := mockGenerateShippingNoticeMixedProducts()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessShippingNoticeGTINsBadRequest(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	inputData := mockGenerateShippingNoticeGTINs()
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError == nil {
		t.Errorf("Expected error")
	}

}

func TestProcessShippingNoticeGTINMaxs(t *testing.T) {
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to '/skus', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
			jsonData, _ = json.Marshal(result)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	config.AppConfig.BatchSizeMax = 1
	inputData := mockGenerateShippingNoticeGTINs()
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError != nil {
		t.Errorf("Error processing shipping notice %s", shippingError)
	}

}

func TestProcessShippingNoticeGTINMaxsBadRequest(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")
	config.AppConfig.EpcToWrin = false
	config.AppConfig.BatchSizeMax = 1
	inputData := mockGenerateShippingNoticeGTINs()
	notificationChan := make(chan alert.Notification, config.AppConfig.NotificationChanSize)
	shippingError := skuMapping.processShippingNotice(&inputData, notificationChan)
	if shippingError == nil {
		t.Errorf("Expected error.")
	}

}

func TestMakeGetCallToSkuMappingWithError(t *testing.T) {
	skuMapping := NewSkuMapping("/skus")
	_, err := MakeGetCallToSkuMapping("", skuMapping.url)
	if err == nil {
		t.Errorf("Expected error.")
	}

}

func TestMakeGetCallToSkuMappingWithMashallError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(1 * time.Second)
		if request.URL.EscapedPath() != "/skus" {
			t.Errorf("Expected request to '/skus', received %s", request.URL.EscapedPath())
		}
		var jsonData []byte
		if request.URL.EscapedPath() == "/skus" {
			//result := buildProductData(0.0, 0.0, 0.0, 0.0, "00111111")
			jsonData, _ = json.Marshal("this")
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(jsonData)
	}))

	defer testServer.Close()

	skuMapping := NewSkuMapping(testServer.URL + "/skus")

	_, errorCall := MakeGetCallToSkuMapping("", skuMapping.url)
	if errorCall == nil {
		t.Errorf("Error expected")
	}

}

func mockGenerateHeartbeat() []byte {
	heartbeat := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
		"value": {
		  "device_id": "rrpgw",
		  "facilities": [
				"facility1",
				"facility2"
		  ],
		  "facility_groups_cfg": "auto-0802233641",
		  "mesh_id": null,
		  "mesh_node_id": null,
		  "personality_groups_cfg": null,
		  "schedule_cfg": "UNKNOWN",
		  "schedule_groups_cfg": null,
		  "sent_on": 1503700192960
		}
	}`)
	return heartbeat
}

func mockGenerateEmptyShippingNoticeWRINs() []byte {
	shippingNotice := []byte(`{
  			"macaddress": "02:42:0a:00:1e:1a",
  			"application": "productmasterdataservicewithdropbox",
  			"providerId": -1,
  			"dateTime": "2018-07-30T19:07:10.461Z",
  			"type": "urn:x-intel:context:retailsensingplatform:shippingmasterdata",
  			"value": {
    			"data": []
  			}
	}`)
	return shippingNotice
}

func mockGenerateEmptyShippingNoticeGTINs() []byte {
	shippingNotice := []byte(`{
  			"macaddress": "02:42:0a:00:1e:1a",
  			"application": "productmasterdataservicewithdropbox",
  			"providerId": -1,
  			"dateTime": "2018-07-30T19:07:10.461Z",
  			"type": "urn:x-intel:context:retailsensingplatform:shippingmasterdata",
  			"value": {
    			"data": []
  			}
	}`)
	return shippingNotice
}

func mockGenerateShippingNoticeProprietaryIDs() []byte {
	shippingNotice := []byte(`{
  			"macaddress": "02:42:0a:00:1e:1a",
  			"application": "productmasterdataservicewithdropbox",
  			"providerId": -1,
  			"dateTime": "2018-07-30T19:07:10.461Z",
  			"type": "urn:x-intel:context:retailsensingplatform:shippingmasterdata",
  			"value": {
    			"data": [
      			{
					"asnId": "AS876422",
					"eventTime": "2018-03-12T12: 34: 56.789Z",
					"siteId": "0105",
					"items": [
					{
						"itemId": "12879047",
						"itemGtin": "00000012879047",
						"itemEpcs": [
							"993402662C00000012879047",
							"993402662C3A500012879047",
							"993402662C3B500012879047",
							"993402662C3C500012879047",
							"993402662C3D500012879047"
							]
					}
					],
					"orderIds": [
						"4500076",
						"4500036"
					]
						
      			},
				{
					"asnId": "AS876423",
					"eventTime": "2018-03-12T12: 59: 56.789Z",
					"siteId": "0105",
					"items": [
					{
						"itemId": "12879048",
						"itemGtin": "00000012879048",
						"itemEpcs": [
							"993402662C00000012879048",
							"993402662C3A500012879048",
							"993402662C3B500012879048",
							"993402662C3C500012879048",
							"993402662C3D500012879048"
							]
					}
					],
					"orderIds": [
						"4500076",
						"4500036"
					]
						
      			}
    			]
  			}
	}`)
	return shippingNotice
}

func mockGenerateShippingNoticeGTINs() []byte {
	shippingNotice := []byte(`{
  			"macaddress": "02:42:0a:00:1e:1a",
  			"application": "productmasterdataservicewithdropbox",
  			"providerId": -1,
  			"dateTime": "2018-07-30T19:07:10.461Z",
  			"type": "urn:x-intel:context:retailsensingplatform:shippingmasterdata",
  			"value": {
    			"data": [
      			{
					"asnId": "AS876422",
					"eventTime": "2018-03-12T12: 34: 56.789Z",
					"siteId": "0105",
					"items": [
						{
							"itemId": "00614141007349",
							"itemGtin": "614141007349",
							"itemEpcs": [
								"3034257BF400B7800004CB2F"
							]
						}
					],
					"orderIds": [
						"4500076"
					]
      			}
    			]
  			}
	}`)
	return shippingNotice
}

func mockGenerateShippingNoticeMixedProducts() []byte {
	shippingNotice := []byte(`{
  			"macaddress": "02:42:0a:00:1e:1a",
  			"application": "productmasterdataservicewithdropbox",
  			"providerId": -1,
  			"dateTime": "2018-07-30T19:07:10.461Z",
  			"type": "urn:x-intel:context:retailsensingplatform:shippingmasterdata",
  			"value": {
    			"data": [
      			{
					"asnId": "AS876422",
					"eventTime": "2018-03-12T12: 34: 56.789Z",
					"siteId": "0105",
					"items": [
					{
						"itemId": "12879047",
						"itemGtin": "00000012879047",
						"itemEpcs": [
							"993402662C00000012879047",
							"993402662C3A500012879047",
							"993402662C3B500012879047",
							"993402662C3C500012879047",
							"993402662C3D500012879047"
							]
					}
					],
					"orderIds": [
						"4500076",
						"4500036"
					]
						
      			},
				{
					"asnId": "AS876422",
					"eventTime": "2018-03-12T12: 34: 56.789Z",
					"siteId": "0105",
					"items": [
						{
							"itemId": "00614141007349",
							"itemGtin": "614141007349",
							"itemEpcs": [
								"3034257BF400B7800004CB2F"
							]
						}
					],
					"orderIds": [
						"4500076"
					]
      			}
    			]
  			}
	}`)
	return shippingNotice
}

func mockGenerateHeartbeatNoFacility() []byte {
	heartbeat := []byte(`{
		"macaddress": "02:42:ac:1d:00:04",
		"application": "rsp_collector",
		"providerId": -1,
		"dateTime": "2017-08-25T22:29:23.816Z",
		"type": "urn:x-intel:context:retailsensingplatform:heartbeat",
		"value": {
		  "device_id": "rrpgw",
		  "facilities": [],
		  "facility_groups_cfg": null,
		  "mesh_id": null,
		  "mesh_node_id": null,
		  "personality_groups_cfg": null,
		  "schedule_cfg": "UNKNOWN",
		  "schedule_groups_cfg": null,
		  "sent_on": 1503700192960
		}
	}`)
	return heartbeat
}

func generateHeartbeatModel(input []byte) (models.Heartbeat, error) {
	var heartbeatEvent models.HeartbeatMessage
	err := json.Unmarshal(input, &heartbeatEvent)
	if err != nil {
		log.Fatalf("error parsing Heartbeat: %s", err)
		return heartbeatEvent.Value, err
	}

	return heartbeatEvent.Value, nil
}

func buildProductData(becomingReadable float64, beingRead float64, dailyTurn float64, exitError float64, productID string) models.SkuMappingResponse {
	var metadata = make(map[string]interface{})
	metadata["becoming_readable"] = becomingReadable
	metadata["being_read"] = beingRead
	metadata["daily_turn"] = dailyTurn
	metadata["exit_error"] = exitError

	productMetadata := models.ProductMetadata{
		ProductID: productID,
	}

	productList := []models.ProductMetadata{productMetadata}

	var data = models.ProdData{
		ProductList: productList,
	}

	dataList := []models.ProdData{data}

	var result = models.SkuMappingResponse{
		ProdData: dataList,
	}
	return result
}

func TestParseReading(t *testing.T) {
	read := edgex.Reading{
		Device: "rrs-gateway",
		Origin: 1471806386919,
		Value:  "{\"jsonrpc\":\"2.0\",\"topic\":\"rfid/gw/heartbeat\",\"params\":{} }",
	}

	reading := parseReadingValue(&read)

	if reading.Topic != "rfid/gw/heartbeat" {
		t.Error("Error parsing Reading Value")
	}
}

func TestParseEvent(t *testing.T) {

	eventStr := `{"origin":1471806386919,
	"device":"rrs-gateway",
	"readings":[ {"name" : "gwevent", "value": " " } ] 
   }`

	event := parseEvent(eventStr)

	if event.Device != "rrs-gateway" || event.Origin != 1471806386919 {
		t.Error("Error parsing edgex event")
	}

}
