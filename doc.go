// RFID Alert Service API
//
// RFID Alert service provides the capabilities to monitor gateway status and alerts. Gateway status is updated periodically based on events sent through Heartbeat listener from CS SDK and posted to the configured REST endpoint.
// In a similar way, alert events are processed and posted to the REST endpoint.
//
//  __Configuration Values__
// <blockquote>RFID Alert service configuration is split between values set in a configuration file and those set as environment values in compose file. The configuration file is expected to be contained in a docker secret for production deployments, but can be on a docker volume for validation and development.
// <blockquote><b>Configuration file values</b>
// <blockquote>•<b> serviceName</b> - Runtime name of the service.</blockquote>
// <blockquote>•<b> loggingLevel</b> - Logging level to use: "info" (default) or "debug" (verbose).</blockquote>
// <blockquote>•<b> port</b> - Port to run the service's HTTP Server on.</blockquote>
// <blockquote>•<b> watchdogMinutes</b> - Time interval set to check the status of registered gateways</blockquote>
// <blockquote>•<b> maxMissedHeartbeats</b> - Maximum heart beats that can be missed before the gateway gets deregistered.</blockquote>
// <blockquote>•<b> sendHeartbeatTo</b> - Endpoint to send processed heart beat.</blockquote>
// <blockquote>•<b> sendAlertTo</b> - Endpoint to send processed alert.</blockquote>
// <blockquote>•<b> contextSensing</b> - Host and port number for the Context Broker.</blockquote>
// </blockquote>
// <blockquote><b>Compose file environment variable values</b>
// <blockquote>•<b> runtimeConfigPath</b> - Path to the configuration file to use at runtime.</blockquote>
// </blockquote>
//
// <pre><b>Example configuration file json
// &#9{
// &#9&#9"serviceName": "RRP - RFID Alert service",
// &#9&#9"loggingLevel": "debug",
// &#9&#9"port": "8080",
// &#9&#9"watchdogMinutes" : 1,
// &#9&#9"maxMissedHeartbeats" : 2,
// &#9&#9"contextSenisng" : "127.0.0.1:8888",
// &#9&#9"sendHeartBeatTo" : "http://rrsnotification:9005/heartbeat",
// &#9&#9"sendAlertTo" : "http://rrsnotification:9005/alert",
// &#9}
// </b></pre>
// <pre><b>Example environment variables in compose file
// &#9runtimeConfigPath: "/data/configs/rfid-alert.json"
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
