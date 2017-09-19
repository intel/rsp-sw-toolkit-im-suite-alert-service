package broker

import (
	"context_linux_go/core"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	contextRPCVersion = "0.9.0"
	jsonRPCVersion    = "2.0"
)

var wsDial = websocket.Dial

// WSBroker represents a websocket broker connection instance
type WSBroker struct {
	itemChan         core.ProviderItemChannel
	errChan          core.ErrorChannel
	application      string
	macAddress       string
	server           string
	connectionPrefix string
	requestID        int
	serverConnection *websocket.Conn
	keepAlive        bool
	doneChan         chan bool
	responseHandlers map[int]interface{}
	CommandHandlers  map[int]core.CommandHandlerInfo //TODO FIX

	retries       int
	retryInterval int

	respHandlerMutex sync.Mutex
}

// NewWebSocketsBroker creates a new broker connection using websockets
func NewWebSocketsBroker(options core.SensingOptions, onItem core.ProviderItemChannel, onErr core.ErrorChannel) *WSBroker {
	var broker WSBroker

	if options.Secure {
		broker.connectionPrefix = "wss"
	} else {
		broker.connectionPrefix = "ws"
	}

	broker.application = options.Application
	broker.macAddress = options.Macaddress
	broker.server = options.Server

	broker.server = options.Server
	broker.itemChan = onItem
	broker.errChan = onErr
	broker.retries = options.Retries
	broker.retryInterval = options.RetryInterval

	broker.responseHandlers = make(map[int]interface{})
	broker.CommandHandlers = make(map[int]core.CommandHandlerInfo)

	return &broker
}

// Publish sends a new item to the broker to be distributed
func (broker *WSBroker) Publish(item *core.ItemData) {
	pubRequest := serverRequestBody{
		Owner: owner{
			Device: device{
				Id: broker.macAddress + ":" + broker.application,
			},
		},
		States: []state{
			state{
				Type:     item.Type,
				DateTime: item.DateTime,
				Value:    core.JSONSchema{"value": item.Value},
			},
		},
	}

	broker.sendToServer("PUT", "states", "", pubRequest, nil)
}

func (broker *WSBroker) getDeviceInfo() deviceInfo {
	macAddress := ""
	interfaces, _ := net.Interfaces()
	for _, item := range interfaces {
		if len(item.HardwareAddr) > 0 {
			macAddress = item.HardwareAddr.String()
			//TODO: Validate MAC Address
			break
		}
	}

	if macAddress == "" {
		broker.errChan <- core.ErrorData{Error: errors.New("MAC Address is Empty")}
	}

	loc := make(map[string]location)
	loc["semantic"] = location{
		Country: "US",
		State:   "OR",
		City:    "Hillsboro",
	}

	devInfo := deviceInfo{
		MacAddress: macAddress,
		Sensors:    nil,
		Name:       broker.application,
		Location:   loc,
	}

	return devInfo
}

func splitDeviceId(deviceId string) (string, string) {
	macaddress := strings.Join(strings.Split(deviceId, ":")[:6], ":")
	application := strings.Split(deviceId, ":")[6]

	return macaddress, application
}

// EstablishConnection tries to connect to the broker and returns an error if it is
// unsuccessful
func (broker *WSBroker) EstablishConnection() error {
	serverURL := fmt.Sprintf("%s://%s/context/v1/socket", broker.connectionPrefix, broker.server)

	config, err := websocket.NewConfig(serverURL, "http://localhost/")
	if err != nil {
		return err
	}
	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	for i := 1; i <= broker.retries+1; i++ {
		connection, err := wsDial(serverURL, "", "http://localhost/")
		if err != nil || connection == nil {
			fmt.Printf("Websocket connection error. Attempt: %v / %v\n", i, broker.retries+1)
			time.Sleep(time.Duration(broker.retryInterval) * time.Second)
		} else {
			broker.serverConnection = connection
			break
		}
	}
	return err
}

// RegisterDevice registers this application with the broker, and calls onStartedHandler
// once the connection has been established
func (broker *WSBroker) RegisterDevice(onStartedHandler interface{}) {

	deviceInfo := broker.getDeviceInfo()
	broker.macAddress = deviceInfo.MacAddress

	go broker.webSocketListener()

	registrationResponseHandler := func(response serverMessage) {
		if response.Error != nil {
			broker.errChan <- core.ErrorData{Error: errors.New("Registration error!")}
		}

		if onStartedHandler != nil {
			handler := reflect.ValueOf(onStartedHandler)
			handler.Call(nil)
		}
	}

	broker.sendToServer("PUT", "devices", "", deviceInfo, registrationResponseHandler)

}

// IsConnected returns true if there is an active connection to the broker
func (broker *WSBroker) IsConnected() bool {
	return broker.serverConnection != nil
}

// RegisterType registers a provider schema with the broker to perform validation
func (broker *WSBroker) RegisterType(schema core.JSONSchema) {
	broker.sendToServer("PUT", "typescatalog", "", schema, nil)
}

// Close ends the connection to the broker
func (broker *WSBroker) Close() {
	broker.doneChan = make(chan bool)
	broker.keepAlive = false

	if broker.serverConnection != nil {
		broker.serverConnection.Close()
		broker.serverConnection = nil

		// Wait until the listener goroutine is done
		<-broker.doneChan
	}
}

func (broker *WSBroker) SetCommandHandler(handlerId int, info core.CommandHandlerInfo) {
	broker.CommandHandlers[handlerId] = info
}

func (broker *WSBroker) getPublishedCommandHandlerIDs(method string) ([]int, error) {
	var handlerIds []int
	for handlerId := range broker.CommandHandlers {
		_, methodTypeFound := broker.CommandHandlers[handlerId].Methods[method]
		if broker.CommandHandlers[handlerId].Publish == true && methodTypeFound {
			handlerIds = append(handlerIds, handlerId)
		}
	}
	if len(handlerIds) == 0 {
		return nil, errors.New("No command handlers found.")
	} else {
		return handlerIds, nil
	}
}
func (broker *WSBroker) sendCommandRPC(method string, params []interface{}, responseHandler interface{}, macAddress string, application string) {
	broker.respHandlerMutex.Lock()
	defer broker.respHandlerMutex.Unlock()

	if macAddress == "" {
		macAddress = "0:0:0:0:0:0"
	}
	if application == "" {
		application = "sensing"
	}

	if responseHandler != nil {
		broker.responseHandlers[broker.requestID] = responseHandler
	}

	requestID := broker.requestID

	jsonStr := serverMessage{
		ContextRPC: contextRPCVersion,
		JsonRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: &serverParams{
			Body: params,
		},
		Id: &requestID,
	}

	broker.requestID++

	err := websocket.JSON.Send(broker.serverConnection, jsonStr)
	if err != nil {
		broker.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}

func (broker *WSBroker) sendCommand(method string, params serverParams, responseHandler interface{}, macAddress string, application string) {
	broker.respHandlerMutex.Lock()
	defer broker.respHandlerMutex.Unlock()

	if macAddress == "" {
		macAddress = "0:0:0:0:0:0"
	}
	if application == "" {
		application = "sensing"
	}

	if responseHandler != nil {
		broker.responseHandlers[broker.requestID] = responseHandler
	}

	requestID := broker.requestID

	jsonStr := serverMessage{
		ContextRPC: contextRPCVersion,
		JsonRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: &params,
		Id:     &requestID,
	}

	broker.requestID++

	err := websocket.JSON.Send(broker.serverConnection, jsonStr)
	if err != nil {
		broker.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}

func (broker *WSBroker) sendToServer(method string, handler string, query string, body interface{}, responseHandler interface{}) {

	params := serverParams{
		Handler: handler,
		Headers: socketHeader{
			Authorization: "Bearer null",
		},
		Body: body,
	}

	if query == "" {
		params.Query = nil
	} else {
		params.Query = &query
	}

	broker.sendCommand(method, params, responseHandler, "", "")
}

func (broker *WSBroker) webSocketListener() {

	broker.keepAlive = true

	for broker.keepAlive {

		if broker.serverConnection == nil {
			broker.errChan <- core.ErrorData{Error: errors.New("Websocket Error - Connection Does Not Exist!")}
			break
		}
		message := serverMessage{}
		err := websocket.JSON.Receive(broker.serverConnection, &message)

		if err != nil {
			broker.errChan <- core.ErrorData{Error: errors.New(fmt.Sprintf("Websocket Error - Received Message: %s", err))}
			if broker.serverConnection != nil {
				broker.serverConnection.Close()
			}
			break
		}

		if isResponse(&message) {
			broker.handleResponse(&message)
		} else {
			broker.handleRequest(&message)
		}
	}

	broker.doneChan <- true
}

func (broker *WSBroker) handleRequest(message *serverMessage) {
	if strings.HasPrefix(*message.Method, "RPC:") {
		broker.handleRPCRequest(message)
		return
	}

	if (message.Endpoint.MacAddress == "0:0:0:0:0:0") && (message.Endpoint.Application == "sensing") && (*message.Method == "PUT") && (message.Params.Handler == "states") {
		body := serverRequestBody{}

		bytes, err := json.Marshal(message.Params.Body)
		if err != nil {
			broker.errChan <- core.ErrorData{Error: errors.New("Error on JSON marshall")}
		}
		err = json.Unmarshal(bytes, &body)
		if err != nil {
			broker.errChan <- core.ErrorData{Error: errors.New("Error on JSON unmarshall")}
		}

		bytes, err = json.Marshal(body.Owner.Device.Id)
		json.Unmarshal(bytes, &body)

		macaddress, application := splitDeviceId(body.Owner.Device.Id)

		for _, stateItem := range body.States {
			itemData := core.ItemData{
				MacAddress:  macaddress,
				Application: application,
				ProviderId:  -1,
				DateTime:    stateItem.DateTime,
				Type:        stateItem.Type,
				Value:       stateItem.Value,
			}

			broker.itemChan <- &itemData
		}
	}
}

func (broker *WSBroker) handleResponse(message *serverMessage) {
	broker.respHandlerMutex.Lock()
	defer broker.respHandlerMutex.Unlock()

	var responseId int
	var responseIdFloat float64
	var responseIdString string
	var splitResponseId []string
	var err error
	var ok bool
	var msg serverMessage = *message
	var messageId interface{} = msg.Id
	responseIdFloat, ok = messageId.(float64)
	if !ok {
		responseIdString, ok = messageId.(string)
		if !ok {
			panic("message ID was not a string or int from broker")
		}
		splitResponseId = strings.Split(responseIdString, ":")
		responseId, err = strconv.Atoi(splitResponseId[len(splitResponseId)-1])
		if err != nil {
			panic("could not convert message ID to int")
		}
	} else {
		responseId = int(responseIdFloat)
	}
	responseHandler := broker.responseHandlers[responseId]
	if responseHandler != nil {
		handler := reflect.ValueOf(responseHandler)
		var handlerArgs []reflect.Value
		handlerArgs = append(handlerArgs, reflect.ValueOf(*message))
		handler.Call(handlerArgs)
	}
	delete(broker.responseHandlers, responseId)
}

func (broker *WSBroker) SendCommand(macAddress string, application string, handlerId int, method string, params []interface{}, valueChannel chan interface{}) {
	if macAddress == "" || application == "" {
		//local handler
		localHandlerInfo, ok := broker.CommandHandlers[handlerId]
		if !ok {
			broker.errChan <- core.ErrorData{Error: errors.New("Unknown Handler: " + strconv.Itoa(handlerId))}
			valueChannel <- core.ErrorData{Error: errors.New("Unknown Handler: " + strconv.Itoa(handlerId))}

		} else {
			localHandlerMethod := reflect.ValueOf(localHandlerInfo.Methods[method])
			var handlerArgs []reflect.Value
			if params != nil && len(params) > 0 {
				for _, param := range params {
					handlerArgs = append(handlerArgs, reflect.ValueOf(param))
				}
			}
			returnedValueArray := localHandlerMethod.Call(handlerArgs)
			if len(returnedValueArray) == 1 {
				returnValue := returnedValueArray[0].Interface()
				valueChannel <- returnValue
			} else {
				returnValue := make([]interface{}, len(returnedValueArray))
				for idx, val := range returnedValueArray {
					returnValue[idx] = val.Interface()
				}
				valueChannel <- returnValue
			}
		}
	} else {
		//remote handler
		sendCommandResponseHandler := func(response serverMessage) {
			if response.Error != nil {
				broker.errChan <- core.ErrorData{Error: errors.New("SendCommand Error!")}
				valueChannel <- core.ErrorData{Error: errors.New("SendCommand Error!")}
				return
			}
			valueChannel <- response.Result.Body
		}
		broker.sendCommandRPC("RPC:"+method, params, sendCommandResponseHandler, macAddress, application)
	}
}

func (broker *WSBroker) handleRPCRequest(request *serverMessage) {
	methodType := strings.Trim(*request.Method, "RPC:")
	//TODO getItem
	handlerIds, err := broker.getPublishedCommandHandlerIDs(methodType)
	if err != nil {
		broker.sendRPCResponse(request, nil, &result{Body: "Method unavailable", ResponseCode: 404})
		return
	}
	for handlerId := range handlerIds {
		valueChannel := make(chan interface{}, 1)
		arrayParams, ok := request.Params.Body.([]interface{})
		if !ok {
			broker.sendRPCResponse(request, nil, &result{Body: "Cannot parse params as array", ResponseCode: 404})
			return
		}
		broker.SendCommand("", "", handlerId, methodType, arrayParams, valueChannel)
		genericValue := <-valueChannel
		//is response?
		response, ok := genericValue.(serverMessage)
		if ok {
			broker.sendRPCResponse(request, response.Result, response.Error)
			return
		}
		//is local error?
		errorValue, ok := genericValue.(error)
		if ok {
			broker.sendRPCResponse(request, nil, &result{Body: errorValue.Error(), ResponseCode: 404})
			return
		}
		broker.sendRPCResponse(request, &result{Body: genericValue, ResponseCode: 200}, nil)
	}
}

func (broker *WSBroker) sendRPCResponse(request *serverMessage, result *result, errorResult *result) {
	messageId := request.Id
	var responseId int
	var err error
	responseEndpoint := request.Endpoint
	responseIdFloat, ok := messageId.(float64)
	if !ok {
		responseIdString, ok := messageId.(string)
		if !ok {
			panic("sending response message ID was not a string or int from broker")
		}
		splitResponseId := strings.Split(responseIdString, ":")
		responseEndpoint = &endpoint{MacAddress: splitResponseId[0] + ":" + splitResponseId[1] + ":" + splitResponseId[2] + ":" + splitResponseId[3] + ":" + splitResponseId[4] + ":" + splitResponseId[5],
			Application: splitResponseId[6]}
		responseId, err = strconv.Atoi(splitResponseId[len(splitResponseId)-1])
		if err != nil {
			panic("sending response could not convert message ID to int")
		}
	} else {
		responseId = int(responseIdFloat)
	}
	response := serverMessage{
		ContextRPC: contextRPCVersion,
		Id:         responseId,
		JsonRPC:    jsonRPCVersion,
		Method:     request.Method,
		Endpoint:   responseEndpoint,
	}
	if errorResult != nil {
		response.Error = errorResult
	} else {
		response.Result = result
	}
	err = websocket.JSON.Send(broker.serverConnection, response)
	if err != nil {
		broker.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}
func isResponse(message *serverMessage) bool {
	if message.Result != nil || message.Error != nil {
		return true
	}
	return false
}
