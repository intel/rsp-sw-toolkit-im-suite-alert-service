package configuration

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"encoding/json"
)

//const defaultConfigFilePath = "./configuration.json"

type Configuration struct {
	parsedJson map[string]interface{}
}

func NewConfiguration() (*Configuration, error) {
	config := Configuration{}

	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		log.Print("No caller information")
	}

	absolutePath := path.Join(path.Dir(filename), "configuration.json")

	// By default load local configuration file if it exists
	if _, err := os.Stat(absolutePath); err != nil {
		absolutePath, ok = os.LookupEnv("runtimeConfigPath")
		if !ok {
			absolutePath = "/run/secrets/configuration.json"
		}
		if _, err := os.Stat(absolutePath); err != nil {
			return nil, fmt.Errorf("specified runtime config file does not exist: %v", err.Error())
		}
	}
	config.Load(absolutePath)
	return &config, nil
}

func (config *Configuration) Load(path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &config.parsedJson)
	if err != nil {
		return err
	}

	return nil
}

func (config *Configuration) GetParsedJson() map[string]interface{} {
	return config.parsedJson
}

func (config *Configuration) GetString(path string) (string, error) {

	if !config.pathExists(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return "", errors.New(fmt.Sprintf("%s not found", path))
		}

		return value, nil
	}

	item := config.getValue(path)

	value, ok := item.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("Unable to convert value for '%s' to a string: Value='%v'", path, item))
	}

	return value, nil
}

func (config *Configuration) GetInt(path string) (int, error) {
	if !config.pathExists(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, errors.New(fmt.Sprintf("%s not found", path))
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("Unable to convert value for '%s' to an int: Value='%v'", path, intValue))
		}

		return intValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, errors.New(fmt.Sprintf("Unable to convert value for '%s' to an int: Value='%v'", path, item))
	}

	return int(value), nil
}

func (config *Configuration) GetFloat(path string) (float64, error) {
	if !config.pathExists(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, errors.New(fmt.Sprintf("%s not found", path))
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("Unable to convert value for '%s' to an int: Value='%v'", path, value))
		}

		return floatValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, errors.New(fmt.Sprintf("Unable to convert value for '%s' to an int: Value='%v'", path, item))
	}

	return value, nil
}

func (config *Configuration) GetBool(path string) (bool, error) {
	if !config.pathExists(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return false, errors.New(fmt.Sprintf("%s not found", path))
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Unable to convert value for '%s' to a bool: Value='%v'", path, boolValue))
		}

		return boolValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(bool)
	if !ok {
		return false, errors.New(fmt.Sprintf("Unable to convert value for '%s' to a bool: Value='%v'", path, item))
	}

	return value, nil
}

func (config *Configuration) GetStringSlice(path string) ([]string, error) {
	if !config.pathExists(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return nil, errors.New(fmt.Sprintf("%s not found", path))
		}

		value = strings.Replace(value, "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		slice := strings.Split(value, ",")
		resultSlice := []string{}
		for _, item := range slice {
			resultSlice = append(resultSlice, strings.Trim(item, " "))
		}

		return resultSlice, nil
	}

	item := config.getValue(path)
	slice := item.([]interface{})

	stringSlice := []string{}
	for _, sliceItem := range slice {
		value, ok := sliceItem.(string)
		if !ok {
			return nil, errors.New(fmt.Sprintf("Unable to convert a value for '%s' to a string: Value='%v'", path, sliceItem))

		}
		stringSlice = append(stringSlice, value)
	}

	return stringSlice, nil
}

func (config *Configuration) getValue(path string) interface{} {
	if config.parsedJson == nil {
		return nil
	}

	pathNodes := strings.Split(path, ".")
	if len(pathNodes) == 0 {
		return nil
	}

	var ok bool
	var value interface{}
	jsonNodes := config.parsedJson
	for _, node := range pathNodes {
		if jsonNodes[node] == nil {
			return nil
		}

		item := jsonNodes[node]
		jsonNodes, ok = item.(map[string]interface{})
		if ok {
			continue
		}

		value = item
		break
	}

	return value
}

func (config *Configuration) pathExists(path string) bool {
	if config.getValue(path) == nil {
		return false
	}

	return true
}
