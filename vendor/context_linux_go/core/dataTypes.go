//Context SDK Golang - v0.9.3

package core

import (
	"fmt"
	"reflect"
)

// Generic JSON schema representation.
// Values of the map can either be strings or nested instances of JSONSchema
type JSONSchema map[string]interface{}

// Represents an item type that a provider produces, including the URN and
// JSON schema
type ProviderType struct {
	URN    string
	Schema JSONSchema
}

// ProviderInterface represents the public API of a local provider.
//
// Some methods are exposed to be used internally within the Sensing core itself
type ProviderInterface interface {
	// Begin producing a stream of items on the onItem channel
	Start(onItem ProviderItemChannel, onErr ErrorChannel)
	// Stop the stream of items to the channel
	Stop()
	// Get the latest item directly without using the item channel.
	// This can either be the last value placed on the channel or a newly created
	// item
	GetItem(string) *ItemData
	// Return the types this Provider can emit. Each package should also expose
	// a Types() function that does not require a ProviderInterface instance to
	// work as well
	Types() []ProviderType
	// Return a pointer to the ProviderOptions struct for this provider.
	// This may be embedded within provider specific options, and is used internally
	// within the Sensing core.
	GetOptions() *ProviderOptions
}
type CommandHandlerInterface interface {
	Start(map[string]interface{})
	Stop()
	Types() []string
	Methods() map[string]interface{} //map of types to method calls
	//MethodDoneChannels() map[string]chan bool

}

type CommandHandlerInfo struct {
	Handler            CommandHandlerInterface
	Methods            map[string]interface{}
	MethodDoneChannels map[string]chan bool
	Publish            bool
}

type SensingOptions struct {
	Application   string
	Macaddress    string
	Server        string
	Secure        bool
	Retries       int
	RetryInterval int
	Publish       bool
	UseCache      bool

	OnStarted SensingStartedChannel
	OnError   ErrorChannel
}

// Common options for all providers. Only Publish should be filled in initially,
// the remaining options will be filled in by the Sensing core during EnableSensing
type ProviderOptions struct {
	// True if the events from this provider should be published to the bus
	Publish bool

	// Identifier of this provider instance, updated during EnableSensing call
	ProviderId int

	// Pointer to the Sensing object to which this provider is attached
	// Must do a type assertion 'Sensing.(sensing.Sensing)' to avoid import cycles
	// Filled in during EnableSensing call
	Sensing interface{}
}

type ItemData struct {
	MacAddress  string      `json:"macaddress"`
	Application string      `json:"application"`
	ProviderId  int         `json:"providerId"`
	DateTime    string      `json:"dateTime"`
	Type        string      `json:"type"`
	Value       interface{} `json:"value"`
}

type ErrorData struct {
	Error error
}

type SensingStarted struct {
	Started bool
}

type ErrorChannel chan ErrorData
type ProviderItemChannel chan *ItemData
type SensingStartedChannel chan *SensingStarted

func (item *ItemData) ToString() string {
	var output string
	output = "{ "
	output += fmt.Sprintf("type: '%s',\n", item.Type)
	output += fmt.Sprintf("value: {")
	val := reflect.ValueOf(item.Value)
	switch val.Kind() {
	case reflect.Map:
		for _, key := range val.MapKeys() {
			output += fmt.Sprintf("%s: %v\n", key, val.MapIndex(key))
		}
	default:
		output += fmt.Sprintf("%v", val)
	}
	output += fmt.Sprintf("},\n")
	output += fmt.Sprintf("macaddress: '%s',\n", item.MacAddress)
	output += fmt.Sprintf("application: '%s',\n", item.Application)
	output += fmt.Sprintf("providerId: '%d',\n", item.ProviderId)
	output += fmt.Sprintf("dateTime: '%s',\n", item.DateTime)
	output += "}"

	return output
}
