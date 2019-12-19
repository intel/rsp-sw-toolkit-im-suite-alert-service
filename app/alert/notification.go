/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package alert

import (
	"net/http"

	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/models"
)

// Notification struct
type Notification struct {
	NotificationType    string
	NotificationMessage string
	Data                interface{}
	GatewayID           string
	Endpoint            string
}

// GeneratePayload wraps the original notification with some additional metadata for use with the Cloud Connector
func (notification *Notification) GeneratePayload() error {
	var payload models.CloudConnectorPayload
	payload.Method = "POST"
	payload.URL = notification.Endpoint
	// Clear the Endpoint property as it was not used in previous versions of the code and it is
	// contained inside of payload.URL
	notification.Endpoint = ""
	header := http.Header{}
	header["Content-Type"] = []string{"application/json"}
	payload.Header = header
	payload.IsAsync = true
	// The original notification data is wrapped inside of the CloudConnectorPayload
	payload.Payload = notification.Data

	// Replace the original notification data with the new CloudConnectorPayload struct
	notification.Data = payload
	return nil
}
