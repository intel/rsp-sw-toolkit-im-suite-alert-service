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

package alert

import (
	"net/http"

	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/config"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/models"
)

// Notification struct
type Notification struct {
	NotificationType    string
	NotificationMessage string
	Data                interface{}
	GatewayID           string
	Endpoint            string
}

// GeneratePayload is to generate cloud connector payload for the alert notification
func (notificationData *Notification) GeneratePayload() error {

	event := notificationData.Data.(models.Alert)

	var payload models.CloudConnectorPayload
	payload.Method = "POST"
	//payload.URL = config.AppConfig.AwsURLHost + config.AppConfig.AwsURLStage + endPoint
	payload.URL = config.AppConfig.AlertDestination
	header := http.Header{}
	header["Content-Type"] = []string{"application/json"}
	payload.Header = header
	payload.IsAsync = true
	payload.Payload = event
	notificationData.Data = payload
	return nil
}
