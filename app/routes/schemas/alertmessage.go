/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package schemas

// AlertMessageSchema is the json schema for alert message
const AlertMessageSchema = `{ 
	"definitions": {
		"Alert": {
		  "required": [
			"sent_on",
			"alert_description",
			"severity"
		  ],
		  "properties": {
			  "alert_description": {
			    "type": "string"
			  },
			  "optional": {
			    "type": "string"
			  },
			  "sent_on": {
			    "type": "integer"
			  },
			  "alert_number": {
				  "type": "integer"
			  },
			  "severity": {
			    "type": "string"
			  },
			  "facilities": {
			    "type": "array"
			  },
			  "device_id": {
			    "type": "string"
			  }
		  },
		  "additionalProperties": false,
		  "type": "object"
		}
	  },
	  "required": [
		"application",
		"value"
	  ],
	  "properties": {
		  "application": {
		    "type": "string"
		  },
		  "macaddress": {
		    "type": "string"
		  },
		  "providerId": {
		    "type": "integer"
		  },
		  "dateTime": {
		    "type": "string"
		  },
		  "value": {
		    "$ref": "#/definitions/Alert"
		  }
	  },
	  "additionalProperties": false,
	  "type": "object"
 }`
