/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"github.com/gorilla/mux"

	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/app/routes/handlers"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/pkg/middlewares"
	"github.com/intel/rsp-sw-toolkit-im-suite-alert-service/pkg/web"
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
