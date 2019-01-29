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

package config

import (
	"github.com/pkg/errors"
	"github.impcloud.net/Responsive-Retail-Core/utilities/configuration"
)

type (
	variables struct {
		ServiceName, LoggingLevel, ContextSensing, Port        string
		NotificationChanSize                                   int
		CloudConnectorURL, CloudConnectorEndpoint              string
		TelemetryDataStoreName, TelemetryEndpoint              string
		WatchdogSeconds                                        int
		MaxMissedHeartbeats                                    int
		SecureMode, SkipCertVerify                             bool
		MappingSkuURL, MappingSkuEndpoint                      string
		EpcToWrin                                              bool
		AlertDestination                                       string
		HeartbeatDestination                                   string
		BatchSizeMax                                           int
		SendNotWhitelistedAlert                                bool
		AlertDestinationAuthEndpoint, AlertDestinationAuthType string
		AlertDestinationClientID, AlertDestinationClientSecret string
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
func InitConfig() error {

	AppConfig = variables{}

	var err error

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ContextSensing, err = config.GetString("contextSensing")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.WatchdogSeconds, err = config.GetInt("watchdogSeconds")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.WatchdogSeconds < 0 {
		return errors.New("Negative value not accepted")
	}

	AppConfig.NotificationChanSize, err = config.GetInt("notificationChanSize")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CloudConnectorURL, err = config.GetString("cloudConnectorURL")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CloudConnectorEndpoint, err = config.GetString("cloudConnectorEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.MaxMissedHeartbeats, err = config.GetInt("maxMissedHeartbeats")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.MaxMissedHeartbeats < 0 {
		return errors.New("Negative value not accepted")
	}

	AppConfig.SecureMode, err = config.GetBool("secureMode")
	if err != nil {
		AppConfig.SecureMode = false
		err = nil
	}

	AppConfig.SkipCertVerify, err = config.GetBool("skipCertVerify")
	if err != nil {
		AppConfig.SkipCertVerify = false
		err = nil
	}

	AppConfig.Port, err = config.GetString("port")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Set "debug" for development purposes. Nil for Production.
	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.MappingSkuURL, err = config.GetString("mappingSkuURL")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	AppConfig.MappingSkuEndpoint, err = config.GetString("mappingSkuEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.EpcToWrin, err = config.GetBool("epcToWrin")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.AlertDestination, err = config.GetString("alertDestination")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.HeartbeatDestination, err = config.GetString("heartbeatDestination")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.BatchSizeMax, err = config.GetInt("batchSizeMax")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SendNotWhitelistedAlert, err = config.GetBool("sendNotWhitelistedAlert")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.AlertDestinationAuthEndpoint, err = config.GetString("alertDestinationAuthEndpoint")
	if err != nil {
		AppConfig.AlertDestinationAuthEndpoint = ""
		err = nil
	}

	AppConfig.AlertDestinationAuthType, err = config.GetString("alertDestinationAuthType")
	if err != nil {
		AppConfig.AlertDestinationAuthType = ""
		err = nil
	}

	AppConfig.AlertDestinationClientID, err = config.GetString("alertDestinationClientID")
	if err != nil {
		AppConfig.AlertDestinationClientID = ""
		err = nil
	}

	AppConfig.AlertDestinationClientSecret, err = config.GetString("alertDestinationClientSecret")
	if err != nil {
		AppConfig.AlertDestinationClientSecret = ""
		err = nil
	}

	return nil
}
