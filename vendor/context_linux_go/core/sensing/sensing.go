//Context SDK Golang - v0.9.3
package sensing

import (
	"context_linux_go/core"
	"context_linux_go/core/broker"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Sensing struct {

	// Local provider
	dispatchers map[int]*localDispatcher
	dispacherID int
	// Synchronize access to the two above fields
	dispMutex sync.Mutex

	// Remote channels
	itemChannels map[string][]*core.ProviderItemChannel
	errChannels  map[string][]*core.ErrorChannel
	// Synchronize add/remove access to the channel maps
	chanMutex sync.Mutex

	//Sensing Vars
	publish          bool
	application      string
	macAddress       string
	server           string
	commandHandlerId int

	// Abstraction for talking to the broker
	broker         broker.Broker
	brokerItemChan core.ProviderItemChannel
	brokerErrChan  core.ErrorChannel

	remoteDone chan bool

	// Array of URNs which have had schemas registered with the broker
	registeredCtxTypes map[string]bool

	// Event cache. String is either providerID:type or mac:application:type like AddContextTypeListener
	eventCache map[string]*core.ItemData
	useCache   bool //Based on options
	cacheMutex sync.Mutex

	//Sensing Channels
	startedC core.SensingStartedChannel
	errC     core.ErrorChannel
}

// Represents a dispatcher for local providers
type localDispatcher struct {
	// Internal for the dispatcher goroutine
	internalItemChannel core.ProviderItemChannel
	internalErrChannel  core.ErrorChannel
	internalStopChannel chan bool

	// From the EnableSensing call. No filter on ContextType
	extItemChannel core.ProviderItemChannel
	extErrChannel  core.ErrorChannel

	// Reference to the provider object
	provider core.ProviderInterface
	// ID of this provider during the EnableSensing call
	providerID int
	// Reference to the sensing object which created this dispatcher
	sensing *Sensing
	// true iff sensing.Publish == true and provider.GetOptions().Publish == true
	publish bool
}

// NewSensing initializes a new instance of the Sensing core. Start() must be called before
// performing other operations
func NewSensing() *Sensing {
	sensing := Sensing{
		dispatchers:    make(map[int]*localDispatcher),
		itemChannels:   make(map[string][]*core.ProviderItemChannel),
		errChannels:    make(map[string][]*core.ErrorChannel),
		remoteDone:     make(chan bool),
		brokerItemChan: make(core.ProviderItemChannel, 10),
		brokerErrChan:  make(core.ErrorChannel, 10),

		registeredCtxTypes: make(map[string]bool),

		eventCache: make(map[string]*core.ItemData),

		broker: &broker.WSBroker{},
	}

	return &sensing
}

// Start begins local and remote sensing using the specified options
func (sensing *Sensing) Start(options core.SensingOptions) {

	if options.Application == "" {
		options.Application, _ = os.Hostname()
	}

	sensing.startedC = options.OnStarted
	sensing.errC = options.OnError

	sensing.server = options.Server

	// TODO need to abstract out the broker further
	sensing.broker = broker.NewWebSocketsBroker(options, sensing.brokerItemChan, options.OnError)

	sensing.publish = options.Publish

	sensing.useCache = options.UseCache

	sensingStarted := func() {
		started := core.SensingStarted{Started: true}
		sensing.startedC <- &started
	}

	// Handle events coming in over the broker item channel.
	// TODO distribute errors?
	go func() {
		for {
			select {
			case item := <-sensing.brokerItemChan:
				specificKey := item.MacAddress + ":" + item.Application + ":" + item.Type
				genericKey := "*:*:" + item.Type
				itemChannels := sensing.itemChannels[specificKey]
				for _, itemChannel := range itemChannels {
					*itemChannel <- item
				}
				itemChannels = sensing.itemChannels[genericKey]
				for _, itemChannel := range itemChannels {
					*itemChannel <- item
				}

				if sensing.useCache {
					sensing.cacheMutex.Lock()
					sensing.eventCache[specificKey] = item
					sensing.eventCache[genericKey] = item
					sensing.cacheMutex.Unlock()
				}
			case <-sensing.remoteDone:
				return
			}
		}
	}()

	if options.Server != "" {
		err := sensing.broker.EstablishConnection()
		if err != nil {
			sensing.errC <- core.ErrorData{Error: err}
		} else if !sensing.broker.IsConnected() {
			sensing.errC <- core.ErrorData{Error: fmt.Errorf(
				"Websocket connection error. Program will exit. Broker address: %v", sensing.server)}
		} else {
			sensing.broker.RegisterDevice(sensingStarted)
		}
	} else {
		sensingStarted()
	}
}

// Stop local and remote sensing operations
func (sensing *Sensing) Stop() {
	sensing.broker.Close()

	// Stop the remote dispatch goroutine
	sensing.remoteDone <- true

	// Stop Local providers
	for id := range sensing.dispatchers {
		sensing.DisableSensing(id)
	}
}

// GetProviders returns a map of enabled local providers
//
// Key: Integer provider ID
//
// Value: Slice of URNs this provider supports
func (sensing *Sensing) GetProviders() map[int][]string {
	result := make(map[int][]string)

	sensing.dispMutex.Lock()
	defer sensing.dispMutex.Unlock()

	for _, disp := range sensing.dispatchers {
		var urns []string
		for _, t := range disp.provider.Types() {
			urns = append(urns, t.URN)
		}
		result[disp.providerID] = urns
	}

	return result
}

// Fill in the local provider item with the common fields before local distribution
// and remote publishing
func (sensing *Sensing) fillInItem(providerID int, item *core.ItemData) {
	if item != nil {
		item.ProviderId = providerID
		item.Application = sensing.application
		item.MacAddress = sensing.macAddress
		item.DateTime = time.Now().Format(time.RFC3339)
	}
}

// Close channels once dispatch is complete for the provider they are registered
// for
func (sensing *Sensing) closeContextTypeChannels(id int) {
	sensing.chanMutex.Lock()
	defer sensing.chanMutex.Unlock()

	strID := fmt.Sprintf("%v", id)

	for key, chs := range sensing.itemChannels {
		// Separate out the provider string from the key
		if strings.SplitN(key, ":", 2)[0] == strID {
			for _, ch := range chs {
				close(*ch)
			}

			delete(sensing.itemChannels, key)
		}
	}

	for key, chs := range sensing.errChannels {
		// Separate out the provider string from the key
		if strings.SplitN(key, ":", 2)[0] == strID {
			for _, ch := range chs {
				close(*ch)
			}
			delete(sensing.errChannels, key)
		}
	}
}

// EnableSensing enables a specific local provider instance. All types this provider produces on its channel
// will be sent to onItem. Any errors will be sent to onErr. The channels may be nil.\
//
// Returns an integer representing the local provider ID
func (sensing *Sensing) EnableSensing(provider core.ProviderInterface, onItem core.ProviderItemChannel, onErr core.ErrorChannel) int {
	sensing.dispMutex.Lock()
	defer sensing.dispMutex.Unlock()

	providerID := sensing.dispacherID + 1
	sensing.dispacherID++

	options := provider.GetOptions()
	options.ProviderId = providerID
	options.Sensing = sensing

	var dispatcher localDispatcher
	dispatcher.internalItemChannel = make(core.ProviderItemChannel, 5)
	dispatcher.internalErrChannel = make(core.ErrorChannel, 5)
	dispatcher.internalStopChannel = make(chan bool)
	dispatcher.extItemChannel = onItem
	dispatcher.extErrChannel = onErr
	dispatcher.provider = provider
	dispatcher.providerID = providerID
	dispatcher.sensing = sensing
	dispatcher.publish = options.Publish && sensing.publish

	sensing.dispatchers[providerID] = &dispatcher

	if dispatcher.publish {
		sensing.registerContextTypesWithBroker(provider)
	}

	// Must be buffered channels in case Start() writes item or err immediately
	// otherwise it will block and never return
	dispatcher.provider.Start(dispatcher.internalItemChannel, dispatcher.internalErrChannel)

	// Flags to see if the provider has closed its channels
	go func() {

		for {
			select {
			case item, ok := <-dispatcher.internalItemChannel:
				if ok {
					sensing.fillInItem(providerID, item)
					dispatcher.dispatch(providerID, item)
				} else {
					dispatcher.internalItemChannel = nil
				}
			case err, ok := <-dispatcher.internalErrChannel:
				if ok {
					dispatcher.dispatch(providerID, err)
				} else {
					dispatcher.internalErrChannel = nil
				}
			case <-dispatcher.internalStopChannel:
				return
			}

			if dispatcher.internalItemChannel == nil && dispatcher.internalErrChannel == nil {
				// Don't close the stop channel, if DisableSensing() is called
				// it will panic on the closed channel
				return
			}
		}
	}()

	return providerID
}

// DisableSensing disables a specific local provider which had been previously enabled with EnableSensing
func (sensing *Sensing) DisableSensing(providerID int) error {
	dispatcher, ok := sensing.dispatchers[providerID]

	// TODO Should send an error on the error channel instead?
	if !ok {
		return fmt.Errorf("Could not find provider for %v", providerID)
	}

	dispatcher.provider.Stop()
	// Signal the goroutine to stop. Shouldn't need to close the channel since we
	// return in the dispatcher select() loop
	dispatcher.internalStopChannel <- true
	delete(sensing.dispatchers, providerID)

	return nil
}

// GetItem polls the provider for the latest item. It can optionally return the latest cached value (if any),
// and update the cache with the new result
func (sensing *Sensing) GetItem(providerID string, contextType string, useCache bool, updateCache bool) *core.ItemData {
	sensing.cacheMutex.Lock()
	defer sensing.cacheMutex.Unlock()

	if useCache {
		if !sensing.useCache {
			sensing.errC <- core.ErrorData{Error: errors.New("Cache not enabled")}
			return nil
		}

		return sensing.eventCache[providerID+":"+contextType]
	}

	if strings.Index(providerID, ":") > 0 {
		// Remote provider
		var remoteItem *core.ItemData // TODO get fresh value from remote
		if updateCache {
			sensing.eventCache[providerID+":"+contextType] = remoteItem
			sensing.eventCache["*.*:"+contextType] = remoteItem
		}
	} else {
		// Query Local provider
		pID, err := strconv.Atoi(providerID)
		if err == nil {
			dispatcher, ok := sensing.dispatchers[pID]
			if ok {
				item := dispatcher.provider.GetItem(contextType)
				sensing.fillInItem(pID, item)
				if updateCache {
					sensing.eventCache[providerID+":"+contextType] = item
				}
				return item
			}

			sensing.errC <- core.ErrorData{Error: fmt.Errorf("Local provider %d not enabled", pID)}
		} else {
			sensing.errC <- core.ErrorData{Error: errors.New("Could not parse provider ID: " + err.Error())}
		}
	}

	return nil
}

// Dispatch a local provider's item to other local providers and optionally
// publish to the bus
func (dispatcher *localDispatcher) dispatch(providerID int, data interface{}) {
	// Local Dispatch
	switch data.(type) {
	case *core.ItemData:
		// EnableSensing path
		item := data.(*core.ItemData)
		if dispatcher.extItemChannel != nil {
			dispatcher.extItemChannel <- item
		}

		// AddContextTypeListener path
		key := fmt.Sprintf("%v:%v", providerID, item.Type)
		for _, channel := range dispatcher.sensing.itemChannels[key] {
			*channel <- item
		}

		// Add item to event cache
		if dispatcher.sensing.useCache {
			dispatcher.sensing.cacheMutex.Lock()
			dispatcher.sensing.eventCache[key] = item
			dispatcher.sensing.cacheMutex.Unlock()
		}

		if dispatcher.publish && dispatcher.sensing.broker.IsConnected() {
			dispatcher.sensing.broker.Publish(item)
		}
	case core.ErrorData:
		err := data.(core.ErrorData)
		// EnableSensing path
		if dispatcher.extErrChannel != nil {
			dispatcher.extErrChannel <- err
		}

		// AddContextTypeListener path
		for _, channel := range dispatcher.sensing.errChannels[fmt.Sprintf("%v", providerID)] {
			*channel <- err
		}

		if dispatcher.publish && dispatcher.sensing.broker.IsConnected() {
			// TODO publish to broker. Node version doesn't do this yet
		}
	default:
		panic(fmt.Sprintf("Unexpected type to dispatch: %T", data))
	}
}

// Register the schema for each context type a provider supports with the broker
// for schema validation
func (sensing *Sensing) registerContextTypesWithBroker(provider core.ProviderInterface) {
	if !sensing.broker.IsConnected() {
		return
	}

	for _, provType := range provider.Types() {
		if _, ok := sensing.registeredCtxTypes[provType.URN]; !ok {
			sensing.broker.RegisterType(provType.Schema)
			sensing.registeredCtxTypes[provType.URN] = true
		}
	}
}

// AddContextTypeListener adds a listener for a specific context type URN. Provider ID can be:
//
// macAddress:Application - for a specific remote provider
//
// *:* - for any remote provider
//
// Stringified integer provider ID - for local providers
func (sensing *Sensing) AddContextTypeListener(providerID string, contextType string, onItem *core.ProviderItemChannel, onErr *core.ErrorChannel) {
	eventKey := providerID + ":" + contextType

	sensing.chanMutex.Lock()
	defer sensing.chanMutex.Unlock()

	if onItem != nil {
		sensing.itemChannels[eventKey] = append(sensing.itemChannels[eventKey], onItem)
	}
	if onErr != nil {
		sensing.errChannels[providerID] = append(sensing.errChannels[providerID], onErr)
	}
}

func (sensing *Sensing) EnableCommands(handler core.CommandHandlerInterface, handlerStartOptions map[string]interface{}) int {
	handlerInfo := core.CommandHandlerInfo{
		Handler: handler,
		Methods: handler.Methods(),
		Publish: false,
	}
	if publish, ok := handlerStartOptions["publish"]; ok {
		handlerInfo.Publish = reflect.ValueOf(publish).Bool()
	}
	sensing.broker.SetCommandHandler(sensing.commandHandlerId, handlerInfo)
	handler.Start(handlerStartOptions)
	returnId := sensing.commandHandlerId
	sensing.commandHandlerId++

	return returnId
}

func (sensing *Sensing) SendCommand(macAddress string, application string, handlerId int, method string, params []interface{}, valueChannel chan interface{}) {
	sensing.broker.SendCommand(macAddress, application, handlerId, method, params, valueChannel)
}
