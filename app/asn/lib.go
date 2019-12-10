/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package asn

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/alert"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/models"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/helper"
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
