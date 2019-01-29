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
	"github.com/pkg/errors"
)

// AdvanceShippingNotice is the model containing advance shipping item epcs
// swagger:model AdvanceShippingNotice
type AdvanceShippingNotice struct {
	AsnID     string                      `json:"asnId"`
	EventTime string                      `json:"eventTime"`
	SiteID    string                      `json:"siteId"`
	Items     []AdvanceShippingNoticeItem `json:"items"`
}

type AdvanceShippingNoticeItem struct {
	Sku       string   `json:"itemId"`
	ProductID string   `json:"itemGtin"`
	Epcs      []string `json:"itemEpcs"`
}

// SkuMappingResponse is the model of the response from the mapping sku service
// with the selection of only the product id
type SkuMappingResponse struct {
	ProdData []ProdData `json:"results"`
}

// ProdData represents the product data schema in the database
type ProdData struct {
	ProductList []ProductMetadata `json:"productList"`
}

// ProductMetadata represents the ProductList schema attribute in the database
type ProductMetadata struct {
	ProductID string `json:"productId"` //product id for now because the skumapping service uses upc and not gtin
}

// ProductID represents the ProductID object
type ProductID struct {
	ProductID string `json:"productId"`
}

// ConvertToASNList convert array string to array of ProductID objects
func ConvertToASNList(asns []string) ([]ProductID, error) {
	if len(asns) == 0 {
		return nil, errors.Errorf("List can't be empty")

	}
	asnList := make([]ProductID, len(asns))
	for i := 0; i < len(asns); i++ {
		asnList[i].ProductID = asns[i]
	}
	return asnList, nil
}
