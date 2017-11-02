package client

import (
	"context_linux_go/core"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	contextRPCVersion  = "0.9.0"
	jsonRPCVersion     = "2.0"
	defaultIPAddress   = "0:0:0:0:0:0"
	defaultApplication = "sensing"
)

// Global so it can be overridden by
var wsDialConfig = websocket.DialConfig

// WSClient represents a websocket wsClient connection instance
type WSClient struct { // nolint: aligncheck
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

	rootCA           *x509.CertPool
	skipVerification bool

	retries       int
	retryInterval int

	respHandlerMutex sync.Mutex
}

// NewWebSocketsClient creates a new wsClient connection using websockets
func NewWebSocketsClient(options core.SensingOptions, onItem core.ProviderItemChannel, onErr core.ErrorChannel) ClientInterface {
	var wsClient WSClient

	if options.Secure {
		wsClient.connectionPrefix = "wss"

		if options.RootCertificates != nil {
			wsClient.rootCA = x509.NewCertPool()
			for _, cert := range options.RootCertificates {
				wsClient.rootCA.AddCert(cert)
			}
		}

		wsClient.skipVerification = options.SkipCertificateVerification
	} else {
		wsClient.connectionPrefix = "ws"
	}

	wsClient.application = options.Application
	wsClient.macAddress = options.Macaddress
	wsClient.server = options.Server

	wsClient.server = options.Server
	wsClient.itemChan = onItem
	wsClient.errChan = onErr
	wsClient.retries = options.Retries
	wsClient.retryInterval = options.RetryInterval

	wsClient.responseHandlers = make(map[int]interface{})
	wsClient.CommandHandlers = make(map[int]core.CommandHandlerInfo)

	// If the connection loop has an error, this channel will be written to before Close()
	wsClient.doneChan = make(chan bool)

	return &wsClient
}

// Publish sends a new item to the wsClient to be distributed
func (wsClient *WSClient) Publish(item *core.ItemData) {
	pubRequest := serverRequestBody{
		Owner: owner{
			Device: device{
				ID: wsClient.macAddress + ":" + wsClient.application,
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

	wsClient.sendToServer("PUT", "states", "", pubRequest, nil)
}

// EstablishConnection tries to connect to the wsClient and returns an error if it is
// unsuccessful
func (wsClient *WSClient) EstablishConnection() error {
	serverURL := fmt.Sprintf("%s://%s/context/v1/socket", wsClient.connectionPrefix, wsClient.server)

	config, err := websocket.NewConfig(serverURL, "http://localhost/")
	if err != nil {
		return err
	}
	config.TlsConfig = &tls.Config{
		RootCAs:            wsClient.rootCA,
		InsecureSkipVerify: wsClient.skipVerification, // nolint: gas
	}

	var connection *websocket.Conn
	for i := 1; i <= wsClient.retries+1; i++ {
		connection, err = wsDialConfig(config)
		if err != nil || connection == nil {
			fmt.Printf("Websocket connection error. Attempt: %v / %v\n", i, wsClient.retries+1)
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Duration(wsClient.retryInterval) * time.Second)

		} else {
			wsClient.serverConnection = connection
			break
		}
	}
	return err
}

// IsConnected returns true if there is an active connection to the wsClient
func (wsClient *WSClient) IsConnected() bool {
	return wsClient.serverConnection != nil
}

// Close ends the connection to the wsClient
func (wsClient *WSClient) Close() {
	wsClient.keepAlive = false

	if wsClient.serverConnection != nil {
		err := wsClient.serverConnection.Close()

		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error closing connection")}
		}
		wsClient.serverConnection = nil

		// Wait until the listener goroutine is done
		<-wsClient.doneChan
	}
}

// SetCommandHandler will set the CommandHandlerInfo object at the array location specified by handlerID
func (wsClient *WSClient) SetCommandHandler(handlerID int, info core.CommandHandlerInfo) {
	wsClient.CommandHandlers[handlerID] = info
}

func (wsClient *WSClient) getPublishedCommandHandlerIDs(method string) ([]int, error) {
	var handlerIDs []int
	for handlerID := range wsClient.CommandHandlers {
		_, methodTypeFound := wsClient.CommandHandlers[handlerID].Methods[method]
		if wsClient.CommandHandlers[handlerID].Publish && methodTypeFound {
			handlerIDs = append(handlerIDs, handlerID)
		}
	}
	if len(handlerIDs) == 0 {
		return nil, errors.New("No command handlers found")
	}

	return handlerIDs, nil
}
func (wsClient *WSClient) sendCommandRPC(method string, params []interface{}, responseHandler interface{}, macAddress string, application string) {
	wsClient.respHandlerMutex.Lock()
	defer wsClient.respHandlerMutex.Unlock()

	if macAddress == "" {
		macAddress = defaultIPAddress
	}
	if application == "" {
		application = defaultApplication
	}

	if responseHandler != nil {
		wsClient.responseHandlers[wsClient.requestID] = responseHandler
	}

	requestID := wsClient.requestID

	jsonStr := serverMessage{
		ContextRPC: contextRPCVersion,
		JSONRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: &serverParams{
			Body: params,
		},
		ID: &requestID,
	}

	wsClient.requestID++

	err := websocket.JSON.Send(wsClient.serverConnection, jsonStr)
	if err != nil {
		wsClient.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}

func (wsClient *WSClient) sendCommand(method string, params serverParams, responseHandler interface{}, macAddress string, application string) {
	wsClient.respHandlerMutex.Lock()
	defer wsClient.respHandlerMutex.Unlock()

	if macAddress == "" {
		macAddress = defaultIPAddress
	}
	if application == "" {
		application = defaultApplication
	}

	if responseHandler != nil {
		wsClient.responseHandlers[wsClient.requestID] = responseHandler
	}

	requestID := wsClient.requestID

	jsonStr := serverMessage{
		ContextRPC: contextRPCVersion,
		JSONRPC:    jsonRPCVersion,
		Method:     &method,
		Endpoint: &endpoint{
			MacAddress:  macAddress,
			Application: application,
		},
		Params: &params,
		ID:     &requestID,
	}

	wsClient.requestID++

	err := websocket.JSON.Send(wsClient.serverConnection, jsonStr)
	if err != nil {
		wsClient.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}

func (wsClient *WSClient) sendToServer(method string, handler string, query string, body interface{}, responseHandler interface{}) {

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

	wsClient.sendCommand(method, params, responseHandler, "", "")
}

func (wsClient *WSClient) webSocketListener() {

	wsClient.keepAlive = true

	for wsClient.keepAlive {

		if wsClient.serverConnection == nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Websocket Error - Connection Does Not Exist")}
			break
		}

		message := serverMessage{}
		err := websocket.JSON.Receive(wsClient.serverConnection, &message)

		if err != nil {
			// If keepAlive is false, we are in the shutdown state and so can recieve errors relating
			// to a closed connection which should be ignored (since there isn't another way to break out of Receive)
			if wsClient.keepAlive {
				wsClient.errChan <- core.ErrorData{Error: fmt.Errorf("Websocket Error - Received Message: %s", err)}
				if wsClient.serverConnection != nil {
					err := wsClient.serverConnection.Close()
					if err != nil {
						wsClient.errChan <- core.ErrorData{Error: errors.New("Error on Server Connection Close")}
					}
				}
			}

			// Break on error always since message is not valid
			break
		}

		if isResponse(&message) {
			wsClient.handleResponse(&message)
		} else {
			wsClient.handleRequest(&message)
		}
	}

	wsClient.doneChan <- true
}

func (wsClient *WSClient) handleRequest(message *serverMessage) { // nolint: gocyclo
	if strings.HasPrefix(*message.Method, "RPC:") {
		wsClient.handleRPCRequest(message)
		return
	}

	if (message.Endpoint.MacAddress == defaultIPAddress) && (message.Endpoint.Application == defaultApplication) && (*message.Method == "PUT") && (message.Params.Handler == "states") {
		body := serverRequestBody{}

		bytes, err := json.Marshal(message.Params.Body)
		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error on JSON marshall")}
		}
		err = json.Unmarshal(bytes, &body)
		if err != nil {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Error on JSON unmarshall")}
		}

		macaddress, application := splitDeviceID(body.Owner.Device.ID)

		for _, stateItem := range body.States {
			itemData := core.ItemData{
				MacAddress:  macaddress,
				Application: application,
				ProviderID:  -1,
				DateTime:    stateItem.DateTime,
				Type:        stateItem.Type,
				Value:       stateItem.Value,
			}

			wsClient.itemChan <- &itemData
		}
	}
}

func (wsClient *WSClient) handleResponse(message *serverMessage) {
	wsClient.respHandlerMutex.Lock()
	defer wsClient.respHandlerMutex.Unlock()

	var responseID int
	var responseIDFloat float64
	var responseIDString string
	var splitResponseID []string
	var err error
	var ok bool
	var msg = *message
	var messageID interface{} = msg.ID
	responseIDFloat, ok = messageID.(float64)
	if !ok {
		responseIDString, ok = messageID.(string)
		if !ok {
			panic("message ID was not a string or int from wsClient")
		}
		splitResponseID = strings.Split(responseIDString, ":")
		responseID, err = strconv.Atoi(splitResponseID[len(splitResponseID)-1])
		if err != nil {
			panic("could not convert message ID to int")
		}
	} else {
		responseID = int(responseIDFloat)
	}
	responseHandler := wsClient.responseHandlers[responseID]
	if responseHandler != nil {
		handler := reflect.ValueOf(responseHandler)
		var handlerArgs []reflect.Value
		handlerArgs = append(handlerArgs, reflect.ValueOf(*message))
		handler.Call(handlerArgs)
	}
	delete(wsClient.responseHandlers, responseID)
}

// SendCommand sends the value to the local or remote handler
func (wsClient *WSClient) SendCommand(macAddress string, application string, handlerID int, method string, params []interface{}, valueChannel chan interface{}) {
	if macAddress == "" || application == "" {
		//local handler
		localHandlerInfo, ok := wsClient.CommandHandlers[handlerID]
		if !ok {
			wsClient.errChan <- core.ErrorData{Error: errors.New("Unknown Handler: " + strconv.Itoa(handlerID))}
			valueChannel <- core.ErrorData{Error: errors.New("Unknown Handler: " + strconv.Itoa(handlerID))}

		} else {
			localHandlerMethod := reflect.ValueOf(localHandlerInfo.Methods[method])
			var handlerArgs []reflect.Value
			if len(params) > 0 {
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
				wsClient.errChan <- core.ErrorData{Error: errors.New("SendCommand Error")}
				valueChannel <- core.ErrorData{Error: errors.New("SendCommand Error")}
				return
			}
			valueChannel <- response.Result.Body
		}
		wsClient.sendCommandRPC("RPC:"+method, params, sendCommandResponseHandler, macAddress, application)
	}
}

func (wsClient *WSClient) handleRPCRequest(request *serverMessage) {
	methodType := strings.Trim(*request.Method, "RPC:")
	//TODO getItem
	handlerIds, err := wsClient.getPublishedCommandHandlerIDs(methodType)
	if err != nil {
		wsClient.sendRPCResponse(request, nil, &result{Body: "Method unavailable", ResponseCode: 404})
		return
	}
	for handlerID := range handlerIds {
		valueChannel := make(chan interface{}, 1)
		arrayParams, ok := request.Params.Body.([]interface{})
		if !ok {
			wsClient.sendRPCResponse(request, nil, &result{Body: "Cannot parse params as array", ResponseCode: 404})
			return
		}
		wsClient.SendCommand("", "", handlerID, methodType, arrayParams, valueChannel)
		genericValue := <-valueChannel
		//is response?
		response, ok := genericValue.(serverMessage)
		if ok {
			wsClient.sendRPCResponse(request, response.Result, response.Error)
			return
		}
		//is local error?
		errorValue, ok := genericValue.(error)
		if ok {
			wsClient.sendRPCResponse(request, nil, &result{Body: errorValue.Error(), ResponseCode: 404})
			return
		}
		wsClient.sendRPCResponse(request, &result{Body: genericValue, ResponseCode: 200}, nil)
	}
}

func (wsClient *WSClient) sendRPCResponse(request *serverMessage, result *result, errorResult *result) {
	messageID := request.ID
	var responseID int
	var err error
	responseEndpoint := request.Endpoint
	responseIDFloat, ok := messageID.(float64)
	if !ok {
		responseIDString, ok := messageID.(string)
		if !ok {
			panic("sending response message ID was not a string or int from wsClient")
		}
		splitResponseID := strings.Split(responseIDString, ":")
		responseEndpoint = &endpoint{MacAddress: splitResponseID[0] + ":" + splitResponseID[1] + ":" + splitResponseID[2] + ":" + splitResponseID[3] + ":" + splitResponseID[4] + ":" + splitResponseID[5],
			Application: splitResponseID[6]}
		responseID, err = strconv.Atoi(splitResponseID[len(splitResponseID)-1])
		if err != nil {
			panic("sending response could not convert message ID to int")
		}
	} else {
		responseID = int(responseIDFloat)
	}
	response := serverMessage{
		ContextRPC: contextRPCVersion,
		ID:         responseID,
		JSONRPC:    jsonRPCVersion,
		Method:     request.Method,
		Endpoint:   responseEndpoint,
	}
	if errorResult != nil {
		response.Error = errorResult
	} else {
		response.Result = result
	}
	err = websocket.JSON.Send(wsClient.serverConnection, response)
	if err != nil {
		wsClient.errChan <- core.ErrorData{Error: errors.New("Websocket send error")}
	}
}
func isResponse(message *serverMessage) bool {
	if message.Result != nil || message.Error != nil {
		return true
	}
	return false
}
