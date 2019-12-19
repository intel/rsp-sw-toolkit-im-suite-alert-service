/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/configuration"
	"github.com/pkg/errors"
)

type (
	variables struct {
		ServiceName, LoggingLevel, Port                        string
		NotificationChanSize                                   int
		CloudConnectorURL, CloudConnectorEndpoint              string
		TelemetryDataStoreName, TelemetryEndpoint              string
		WatchdogSeconds                                        int
		MaxMissedHeartbeats                                    int
		MappingSkuURL, MappingSkuEndpoint                      string
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
