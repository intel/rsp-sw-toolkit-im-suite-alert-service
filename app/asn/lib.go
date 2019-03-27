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

package asn

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/models"
	"time"
)

func buildNotWhitelistedAlert(notWhitelisted []models.ProductID) models.Alert {
	var notWhitelistedAlert models.Alert
	notWhitelistedAlert.SentOn = helper.UnixMilliNow()
	notWhitelistedAlert.AlertDescription = "Received a list of ASNs that are not whitelisted!"
	notWhitelistedAlert.DeviceID = ""
	notWhitelistedAlert.Facilities = []string{}
	notWhitelistedAlert.AlertNumber = alert.NotWhitelisted
	notWhitelistedAlert.Severity = "critical"
	notWhitelistedAlert.Optional = notWhitelisted
	return notWhitelistedAlert
}

func GenerateNotWhitelistedAlert(notWhitelisted []models.ProductID) ([]byte, error) {
	var alertMessage models.AlertMessage
	alert := buildNotWhitelistedAlert(notWhitelisted)

	alertMessage.Application = "advancedshippingnotice"
	alertMessage.Value = alert
	alertMessage.Datetime = time.Now()
	alertMessage.ProviderID = -1
	alertMessage.MACAddress = "00:00:00:00:00:00"

	alertMessageBytes, err := json.Marshal(alertMessage)
	if err != nil {
		return nil, errors.Wrap(err, "Marshaling AlertMessage to []bytes")
	}

	return alertMessageBytes, nil
}
