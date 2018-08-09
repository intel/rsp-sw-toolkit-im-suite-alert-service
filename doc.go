// RFID Alert Service API
//
// RFID Alert service provides the capabilities to monitor gateway status and send alerts. Gateway status is updated periodically based on events sent through Heartbeat listener from CS SDK and alerts are generated when heartbeats are missed.
// And also, alert events are processed and posted to the REST endpoint.
//
//  __Configuration Values__
// <blockquote>RFID Alert service configuration is split between values set in a configuration file and those set as environment values in compose file. The configuration file is expected to be contained in a docker secret for production deployments, but can be on a docker volume for validation and development.
// <blockquote><b>Configuration file values</b>
// <blockquote>•<b> serviceName</b> - Runtime name of the service.</blockquote>
// <blockquote>•<b> loggingLevel</b> - Logging level to use: "info" (default) or "debug" (verbose).</blockquote>
// <blockquote>•<b> notificationChanSize</b> - Channel size of a go channel named as notificationChan.</blockquote>
// <blockquote>•<b> port</b> - Port to run the service's HTTP Server on.</blockquote>
// <blockquote>•<b> contextSensing</b> - Host and port number for the Context Broker.</blockquote>
// <blockquote>•<b> watchdogSeconds</b> - Time interval set to check the status of registered gateways</blockquote>
// <blockquote>•<b> maxMissedHeartbeats</b> - Maximum heart beats that can be missed before the gateway gets deregistered.</blockquote>
// <blockquote>•<b> cloudConnectorURL</b> - URL for Cloud-connector service.</blockquote>
// <blockquote>•<b> cloudConnectorEndpoint</b> - Endpoint for Cloud-connector service.</blockquote>
// <blockquote>•<b> Destination</b> - Endpoint URL for Cloud-connector to send notificaiton to.</blockquote>
// <blockquote>•<b> jwtSignerURL</b> - URL for Jwt-signing service.</blockquote>
// <blockquote>•<b> jwtSignerEndpoint</b> - Endpoint for Jwt-signing service.</blockquote>
// <blockquote>•<b> secureMode</b> - Boolean flag indicating if using secure connection to the Context Brokers.</blockquote>
// <blockquote>•<b> skipCertVerify</b> - Boolean flag indicating if secure connection to the Context Brokers should skip certificate validation.</blockquote>
// <blockquote>•<b> telemetryEndpoint</b> - URL of the telemetry service receiving the metrics from the service.</blockquote>
// <blockquote>•<b> telemetryDataStoreName</b> - Name of the data store in the telemetry service to store the metrics.</blockquote>
// </blockquote>
// <blockquote><b>Compose file environment variable values</b>
// <blockquote>•<b> cloudConnectorURL</b> - URL to send processed heart beat and alerts.</blockquote>
// <blockquote>•<b> runtimeConfigPath</b> - Path to the configuration file to use at runtime.</blockquote>
// <blockquote>•<b> contextSensing</b> - Host and port number for the Context Broker.</blockquote>
// </blockquote>
//
// <pre><b>Example configuration file json
// &#9{
// &#9&#9"serviceName": "RRP - RFID Alert service",
// &#9&#9"loggingLevel": "debug",
// &#9&#9"notificationChanSize": 100,
// &#9&#9"port": "8080",
// &#9&#9"contextSensing": "127.0.0.1:8888",
// &#9&#9"watchdogSeconds": 1,
// &#9&#9"maxMissedHeartbeats": 3,
// &#9&#9"cloudConnectorURL": "http://127.0.0.1:8081",
// &#9&#9"cloudConnectorEndpoint": "/aws/invoke",
// &#9&#9"Destination": "https://test.com/call",
// &#9&#9"jwtSigningURL": "http://127.0.0.1:8080",
// &#9&#9"jwtSigningEndpoint": "/jwt-signing/sign",
// &#9&#9"secureMode": false,
// &#9&#9"skipCertVerify": false,
// &#9&#9"telemetryEndpoint": "http://166.130.9.122:8000",
// &#9&#9"telemetryDataStoreName" : "Store105",
// &#9}
// </b></pre>
// <pre><b>Example environment variables in compose file
// &#9cloudConnectorURL: "https://cloudconnector:5001",
// &#9runtimeConfigPath: "/data/configs/rfid-alert.json"
// &#9contextSenisng: "127.0.0.1:8888",
// </b></pre>
// </blockquote>
//
//	   __Secrets__
// The following values/files are passed to the service via Docker Secrets
//<blockquote>
// <blockquote>•<b> configuration.json</b> - Configuration file referenced above</blockquote>
//</blockquote>
//
// __Known services this service depends on:__
// ○ Context Sensing
// ○ Jwt-signing
// ○ Cloud-connector
//
// __Known services that depend upon this service:__
// ○ None
//
// Schemes: http, https
// Host: rfid-alert-service:8080
//      Contact: RRP <rrp@intel.com>
// BasePath: /
// Version: 0.0.1
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// swagger:meta
package main

// statusOk
//
// swagger:response statusOk
type statusOk struct {
}

// serviceUnavailable
//
// swagger:response serviceUnavailable
type serviceUnavailable struct {
}
