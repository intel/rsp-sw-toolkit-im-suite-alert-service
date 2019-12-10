/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
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
