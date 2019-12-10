/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package models

import "net/http"

// CloudConnectorPayload is the model containing the payload send to webhook url
type CloudConnectorPayload struct {
	URL     string      `json:"url"`
	Header  http.Header `json:"header"`
	Payload interface{} `json:"payload"`
	// Authentication data
	Auth    Auth   `json:"auth"`
	Method  string `json:"method"`
	IsAsync bool   `json:"isasync"`
}

// Auth contains the type and the endpoint of authentication
type Auth struct {
	AuthType string `json:"authtype"`
	Endpoint string `json:"endpoint"`
	Data     string `json:"data"`
}
