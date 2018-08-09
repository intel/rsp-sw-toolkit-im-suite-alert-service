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

package routes

import (
	"github.com/gorilla/mux"

	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/app/routes/handlers"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/middlewares"
	"github.impcloud.net/Responsive-Retail-Inventory/rfid-alert-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter() *mux.Router {

	alerts := handlers.Alerts{}

	var routes = []Route{
		// swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests, i.e is healthy
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			alerts.GetIndex,
		},
		// swagger:route POST /rfid-alert/alertmessage sendAlertMessage
		//
		// Send alert message for events and post the message to the cloud connector
		//
		// Alert message for events should be in request body payload in JSON format.<br><br>
		//
		// Example AlertMessage Input:
		// ```
		// {
		// &#9"application":"Inventory-service",
		// &#9"value": {
		// &#9&#9"sent_on":1531522680000,
		// &#9&#9"alert_description":"Deletion of system database tag collection is complete",
		// &#9&#9"severity":"critical",
		// &#9&#9"optional":"event alert related data"
		// &#9}
		// }
		// ```
		//
		//
		// + application  - the application sending the alert message
		// + sent_on  - the time that alert message is sent in millisecond epoch
		// + alert_description  - the detailed message for the alert
		// + severity  - the severity of the alert
		// + optional  - contains any alert related data or evidence and can be omitted
		//
		//
		//
		// Response will be either Ok or error messages when posting to cloud connector fails.
		//
		//     Consumes:
		//     - application/json
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http
		//
		//     Responses:
		//       200: statusOk
		//       400: schemaValidation
		//       500: internalError
		//       503: serviceUnavailable
		//
		{
			"SendAlertMessage",
			"POST",
			"/rfid-alert/alertmessage",
			alerts.SendAlertMessageToCloudConnector,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {

		var handler = route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
