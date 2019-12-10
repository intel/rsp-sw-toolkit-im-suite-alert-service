/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package asn

import (
	"encoding/json"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-alert-service/app/models"
	"reflect"
	"testing"
)

func TestGenerateNotWhitelistedAlert(t *testing.T) {
	var notWhitelisted = []string{
		"30143639F84191AD22900204",
	}

	asnList, err := models.ConvertToASNList(notWhitelisted)
	if err != nil {
		t.Errorf("error generating alert")
	}
	alert, err := GenerateNotWhitelistedAlert(asnList)
	if err != nil {
		t.Errorf("error generating alert")
	}

	var alertMessage models.AlertMessage

	err = json.Unmarshal(alert, &alertMessage)
	if err != nil {
		t.Errorf("Alert did not unmarshall correctly")
	}
	var a []models.AdvanceShippingNotice
	alertMessageBytes, err := json.Marshal(alertMessage.Value.Optional)
	if err != nil {
		t.Errorf("Marshaling AlertMessage to []bytes")
	}
	json.Unmarshal(alertMessageBytes, &a)

	if alertMessage.Value.Severity != "critical" && !reflect.DeepEqual(a, asnList) {
		t.Errorf("Error creating critical not whitelisted alert")
	}
}
