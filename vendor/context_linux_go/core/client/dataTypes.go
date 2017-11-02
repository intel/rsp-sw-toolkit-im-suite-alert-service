// Package client is an abstraction for connections to the bus
package client

import (
	"context_linux_go/core"
)

// ClientInterface represents an abstract connection to a broker server (bus) which distributes messages
// out to different clients
type ClientInterface interface { // nolint: golint
	EstablishConnection() error
	IsConnected() bool
	Close()

	Publish(item *core.ItemData)
	RegisterDevice(onStartedHandler interface{})
	RegisterType(schema core.JSONSchema)
	SendCommand(macAddress string, application string, handlerID int, method string, params []interface{}, valueChannel chan interface{})
	SetCommandHandler(handlerID int, info core.CommandHandlerInfo)
}

type deviceInfo struct {
	MacAddress string              `json:"MAC"`
	Sensors    *string             `json:"sensors"`
	Name       string              `json:"name"`
	Location   map[string]location `json:"location"`
}

type socketHeader struct {
	Authorization string `json:"Authorization"`
}

type serverParams struct {
	Handler string       `json:"handler"`
	Headers socketHeader `json:"headers"`
	Body    interface{}  `json:"body"`
	Query   interface{}  `json:"query"`
}

type result struct {
	Body         interface{} `json:"body"`
	ResponseCode int         `json:"response_code"`
}

type serverMessage struct {
	ContextRPC string        `json:"contextrpc"`
	JSONRPC    string        `json:"jsonrpc"`
	Method     *string       `json:"method"`
	Endpoint   *endpoint     `json:"endpoint"`
	Params     *serverParams `json:"params"`
	ID         interface{}   `json:"id"`

	Result *result `json:"result"`
	Error  *result `json:"error"`
}

type device struct {
	ID string `json:"id"`
}

type owner struct {
	Device device `json:"device"`
}

type state struct {
	DateTime string      `json:"dateTime"`
	Type     string      `json:"type"`
	Value    interface{} `json:"value"`
}

type serverRequestBody struct {
	Owner  owner   `json:"owner"`
	States []state `json:"states"`
}

type location struct {
	Country string `json:"country"`
	State   string `json:"state"`
	City    string `json:"city"`
}

type endpoint struct {
	MacAddress  string `json:"macaddress"`
	Application string `json:"application"`
}
