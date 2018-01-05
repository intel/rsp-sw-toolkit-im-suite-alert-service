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
	"github.impcloud.net/Responsive-Retail-MVP/utilities/configuration"
)

type (
	variables struct {
		ServiceName, LoggingLevel, ContextSensing string
		SendHeartbeatTo, SendAlertTo, SendEventTo string
		WatchdogMinutes, MaxMissedHeartbeats      int
		SecureMode, SkipCertVerify                bool
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
func InitConfig() error {

	AppConfig = variables{}

	var err error

	var config = *configuration.NewConfiguration()

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ContextSensing, err = config.GetString("contextSensing")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.WatchdogMinutes, err = config.GetInt("watchdogMinutes")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SendHeartbeatTo, err = config.GetString("sendHeartbeatTo")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SendAlertTo, err = config.GetString("sendAlertTo")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
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

	// Set "debug" for development purposes. Nil for Production.
	AppConfig.LoggingLevel = config.GetStringOptional("loggingLevel")

	return nil
}