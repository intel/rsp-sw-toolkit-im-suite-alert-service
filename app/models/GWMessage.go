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

package models

import (
	"time"
)

// HeartBeatMessageValue is used inside of an Alert
type HeartBeatMessageValue struct {
	DeviceID             string   `json:"device_id"`
	Facilities           []string `json:"facilities"`
	FacilityGroupsCfg    string   `json:"facility_groups_cfg"`
	MeshID               string   `json:"mesh_id"`
	MeshNodeID           string   `json:"mesh_node_id"`
	PersonalityGroupsCfg string   `json:"personality_groups_cfg"`
	ScheduleCfg          string   `json:"schedule_cfg"`
	ScheduleGroupsCfg    string   `json:"schedule_groups_cfg"`
	SentOn               int      `json:"sent_on"`
}

// HeartBeatMessage represents an alert that will be sent
type HeartBeatMessage struct {
	MACAddress  string                `json:"macaddress"`
	Application string                `json:"application"`
	ProviderID  int                   `json:"providerId"`
	Datetime    time.Time             `json:"dateTime, string"`
	Details     HeartBeatMessageValue `json:"value"`
}

/*{
  "macaddress": "02:42:ac:1a:00:05",
  "application": "rsp_collector",
  "providerId": -1,
  "dateTime": "2017-09-27T17:16:57.644Z",
  "type": "urn:x-intel:context:retailsensingplatform:heartbeat",
  "value": {
    "device_id": "rsdrrp",
    "facilities": [
      "front"
    ],
    "facility_groups_cfg": "auto-0310080051",
    "mesh_id": null,
    "mesh_node_id": null,
    "personality_groups_cfg": null,
    "schedule_cfg": "UNKNOWN",
    "schedule_groups_cfg": null,
    "sent_on": 1506532617643
  }
}*/

// Schema represents the schema for heartbeat message
const HeartBeatSchema = `
{
  "definitions": {
      "HeartBeatMessageValue": {
          "required": [
              "sent_on",
              "device_id",
              "facilities"
          ],
          "properties": {
              "device_id": {
                  "type": "string",
                  "pattern": "^[-A-Za-z0-9_ \\.]+$"
              },
              "facilities": {
                  "type": "array",
                  "items": {
                      "type": "string"
                  }
              },
              "facility_groups_cfg": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "mesh_id": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "mesh_node_id": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "personality_groups_cfg": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "schedule_cfg": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "schedule_groups_cfg": {
                  "type": [
                      "string",
                      "null"
                  ]
              },
              "sent_on": {
                  "type": "integer"
              }
          },
          "additionalProperties": false,
          "type": "object"
      }
  },
  "type": "object",
  "required": [
      "application"
  ],
  "properties": {
      "application": {
          "type": "string"
      },
      "dateTime": {
          "type": "string",
          "format": "date-time"
      },
      "macaddress": {
          "type": "string"
      },
      "providerId": {
          "type": "integer"
      },
      "type": {
        "type": "string"
    },
      "value": {
          "$ref": "#/definitions/HeartBeatMessageValue"
      }
  },
  "additionalProperties": false
}
`
