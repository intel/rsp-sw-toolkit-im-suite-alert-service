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

package sgtin96

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//noinspection GoNameStartsWithPackageName
const (
	maxPartitionValue = 6
	digitCount        = 13
	numEpcBits        = 96
	numEpcDigits      = numEpcBits / 4   // 4 bits per hex digit
	numEpcBytes       = numEpcDigits / 2 // 2 digits per byte
	sgtin96Header     = "30"
)

var l = [7]int{12, 11, 10, 9, 8, 7, 6} // Number of digits in the company prefix

// GetGtin14 converts the specified EPC String to GTIN-14
// according to the SGTIN-96 format specified by GS1 in the
// EPC Generation 1 Tag Data Standards Version 1.1 Rev.1.27.
// The method throws an Error if the EPC
// is not encoded properly according the GS1 standard.
// nolint: gocyclo
func GetGtin14(epc string) (string, error) {
	if inputErr := checkEpc(epc); inputErr != nil {
		return "", inputErr
	}

	// Convert the EPC string into an array of bytes
	epcBytes, convByteErr := getEpcBytes(epc)
	if convByteErr != nil {
		return "", convByteErr
	}

	// Validate the partition value (v1.1 line 784)
	partitionValue, err := getPartitionValue(epcBytes)
	if err != nil {
		return "", err
	}

	// Validate the company prefix value (v1.1 line 788)
	companyPrefix, err := getCompanyPrefix(epcBytes, partitionValue)
	if err != nil {
		return "", err
	}

	// Validate the Item Reference and Indicator (v1.1 line 792)
	itemReference, err := getItemReference(epcBytes, partitionValue)
	if err != nil {
		return "", err
	}

	// Calculate the GTIN-14 value (v1.1 lines 796 - 800)
	gtin14, err := getCheckDigit(partitionValue, companyPrefix, itemReference)
	if err != nil {
		return "", err
	}

	// Determine the output string
	code := strconv.Itoa(gtin14[0])
	for i := 1; i < 1+l[partitionValue]; i++ {
		code += strconv.Itoa(gtin14[i])
	}
	for i := 1 + l[partitionValue]; i < digitCount; i++ {
		code += strconv.Itoa(gtin14[i])
	}
	code += strconv.Itoa(gtin14[digitCount])

	return code, nil
}

// GetCompanyPrefixByEpc returns company prefix based on hexadecimal EPC string code
func GetCompanyPrefixByEpc(epc string) (int64, error) {
	if inputErr := checkEpc(epc); inputErr != nil {
		return -1, inputErr
	}

	// Convert the EPC string into an array of bytes
	epcBytes, convByteErr := getEpcBytes(epc)
	if convByteErr != nil {
		return -1, convByteErr
	}

	// Validate the partition value (v1.1 line 784)
	partitionValue, err := getPartitionValue(epcBytes)
	if err != nil {
		return -1, err
	}

	// Validate the company prefix value (v1.1 line 788)
	companyPrefix, err := getCompanyPrefix(epcBytes, partitionValue)
	if err != nil {
		return -1, err
	}

	return companyPrefix, nil
}

func checkEpc(epc string) error {
	// Only allow EPC values with an SGTIN-96 header value
	if !strings.HasPrefix(epc, sgtin96Header) {
		return errors.New("EPC has wrong binary header for SGTIN-96")
	}

	// Only allow 96 bit EPC values
	if len(epc) != numEpcDigits {
		return errors.New("EPC wrong length for SGTIN-96 conversion")
	}
	return nil
}

func getEpcBytes(epc string) ([numEpcBytes]byte, error) {
	epcBytes := [numEpcBytes]byte{}
	for i := 0; i < len(epcBytes); i++ {
		tempParse, err := strconv.ParseUint(epc[i*2:(i*2)+2], 16, 8)
		if err != nil {
			return epcBytes, err
		}
		epcBytes[i] = byte(tempParse) & 0xFF
	}
	return epcBytes, nil
}

// GetItemFilter parses out the item filter encoded within the EPC
// 0 - All Others
// 1 - POS Item
// 2 - Case
// 3 - Reserved
// 4 - Inner Pack
// 5 - Reserved
// 6 - Unit Load
// 7 - Component
func GetItemFilter(epc string) (int64, error) {
	if inputErr := checkEpc(epc); inputErr != nil {
		return -1, inputErr
	}
	epcBytes, err := getEpcBytes(epc)
	if err != nil {
		return -1, err
	}
	filterValue := int64(byte(0) | epcBytes[1]>>5)
	return filterValue, nil
}
func getPartitionValue(epc [numEpcBytes]byte) (int, error) {
	partitionValue := int(byte(0) | (epc[1]&0x1C)>>2)
	if partitionValue < 0 || partitionValue > maxPartitionValue {
		return 0, errors.New("invalid partition value for SGTIN-96 conversion")
	}

	return partitionValue, nil
}

//nolint :gocyclo
func getCompanyPrefix(epc [numEpcBytes]byte, partitionValue int) (int64, error) {
	companyPrefix := int64(0)

	// nolint: dupl
	switch partitionValue {
	case 0: // 40 bits
		companyPrefix |= int64(epc[1]&0x03) << 38
		companyPrefix |= int64(epc[2]&0xFF) << 30
		companyPrefix |= int64(epc[3]&0xFF) << 22
		companyPrefix |= int64(epc[4]&0xFF) << 14
		companyPrefix |= int64(epc[5]&0xFF) << 6
		companyPrefix |= int64(epc[6]&0xFC) >> 2
	case 1: // 37 bits
		companyPrefix |= int64(epc[1]&0x03) << 35
		companyPrefix |= int64(epc[2]&0xFF) << 27
		companyPrefix |= int64(epc[3]&0xFF) << 19
		companyPrefix |= int64(epc[4]&0xFF) << 11
		companyPrefix |= int64(epc[5]&0xFF) << 3
		companyPrefix |= int64(epc[6]&0xE0) >> 5
	case 2: // 34 bits
		companyPrefix |= int64(epc[1]&0x03) << 32
		companyPrefix |= int64(epc[2]&0xFF) << 24
		companyPrefix |= int64(epc[3]&0xFF) << 16
		companyPrefix |= int64(epc[4]&0xFF) << 8
		companyPrefix |= int64(epc[5] & 0xFF)
	case 3: // 30 bits
		companyPrefix |= int64(epc[1]&0x03) << 28
		companyPrefix |= int64(epc[2]&0xFF) << 20
		companyPrefix |= int64(epc[3]&0xFF) << 12
		companyPrefix |= int64(epc[4]&0xFF) << 4
		companyPrefix |= int64(epc[5]&0xF0) >> 4
	case 4: // 27 bits
		companyPrefix |= int64(epc[1]&0x03) << 25
		companyPrefix |= int64(epc[2]&0xFF) << 17
		companyPrefix |= int64(epc[3]&0xFF) << 9
		companyPrefix |= int64(epc[4]&0xFF) << 1
		companyPrefix |= int64(epc[5]&0x80) >> 7
	case 5: // 24 bits
		companyPrefix |= int64(epc[1]&0x03) << 22
		companyPrefix |= int64(epc[2]&0xFF) << 14
		companyPrefix |= int64(epc[3]&0xFF) << 6
		companyPrefix |= int64(epc[4]&0xFC) >> 2
	case 6: // 20 bits
		companyPrefix |= int64(epc[1]&0x03) << 18
		companyPrefix |= int64(epc[2]&0xFF) << 10
		companyPrefix |= int64(epc[3]&0xFF) << 2
		companyPrefix |= int64(epc[4]&0xC0) >> 6
	default:
		return 0, errors.New("invalid partition value for SGTIN-96 conversion")
	}

	if companyPrefix < 0 || companyPrefix >= int64(math.Pow(float64(10), float64(l[partitionValue]))) {
		return 0, errors.New("invalid company prefix for SGTIN-96 conversion")
	}

	return companyPrefix, nil
}

//nolint :gocyclo
func getItemReference(epc [numEpcBytes]byte, partitionValue int) (int64, error) {
	var itemReference = int64(0)

	// nolint: dupl
	switch partitionValue {
	case 0: // 4 bits
		itemReference = itemReference | ((int64(epc[6]) & 0x03) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 1: // 7 bits
		itemReference = itemReference | ((int64(epc[6]) & 0x1F) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 2: // 10 bits
		itemReference = itemReference | ((int64(epc[6]) & 0xFF) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 3: // 14 bits
		itemReference = itemReference | ((int64(epc[5]) & 0x0F) << 10)
		itemReference = itemReference | ((int64(epc[6]) & 0xFF) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 4: // 17 bits
		itemReference = itemReference | ((int64(epc[5]) & 0x7F) << 10)
		itemReference = itemReference | ((int64(epc[6]) & 0xFF) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 5: // 20 bits
		itemReference = itemReference | ((int64(epc[4]) & 0x03) << 18)
		itemReference = itemReference | ((int64(epc[5]) & 0xFF) << 10)
		itemReference = itemReference | ((int64(epc[6]) & 0xFF) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	case 6: // 24 bits
		itemReference = itemReference | ((int64(epc[4]) & 0x3F) << 18)
		itemReference = itemReference | ((int64(epc[5]) & 0xFF) << 10)
		itemReference = itemReference | ((int64(epc[6]) & 0xFF) << 2)
		itemReference = itemReference | ((int64(epc[7]) & 0xC0) >> 6)
	default:
		return 0, errors.New("invalid partition value for SGTIN-96 conversion")
	}

	if itemReference < 0 || itemReference >= int64(math.Pow(float64(10), float64(partitionValue+1))) {
		return 0, errors.New("invalid item reference for SGTIN-96 conversion")
	}

	return itemReference, nil
}

func getCheckDigit(partitionValue int, companyPrefix int64, itemReference int64) ([digitCount + 1]int, error) {
	gtin14 := [digitCount + 1]int{}

	var cpNumDigits = l[partitionValue]
	var irNumDigits = digitCount - cpNumDigits

	companyPrefixStr := fmt.Sprintf("%0"+strconv.Itoa(cpNumDigits)+"d", companyPrefix)
	itemReferenceStr := fmt.Sprintf("%0"+strconv.Itoa(irNumDigits)+"d", itemReference)

	// Construct the magical 13-digit number per spec
	// Set digit 1 according to GS1 Tag Data Spec v1.6 line 1129
	tempI, err := strconv.ParseInt(itemReferenceStr[0:1], 10, 64)
	if err != nil {
		return [digitCount + 1]int{}, errors.Wrap(err, "error computing check digit")
	}
	gtin14[0] = int(tempI)

	// Set digit 2 through L+1
	for i := 1; i < cpNumDigits+1; i++ {
		tempI, err := strconv.ParseInt(companyPrefixStr[i-1:i], 10, 64)
		if err != nil {
			return [digitCount + 1]int{}, errors.Wrap(err, "error computing check digit")
		}
		gtin14[i] = int(tempI)
	}

	// Set digit L+2 through 13
	for i := cpNumDigits + 1; i < digitCount; i++ {
		tempI, err := strconv.ParseInt(itemReferenceStr[i-cpNumDigits:i-cpNumDigits+1], 10, 64)
		if err != nil {
			return [digitCount + 1]int{}, errors.Wrap(err, "error computing check digit")
		}
		gtin14[i] = int(tempI)
	}

	// Calculate the check digit per spec (remember the array is zero based)
	// Check digit d14 = (-3 * (d1 + d3 + d5 + d7 + d9 + d11 + d13) - (d2 + d4 + d6 + d8 + d10 + d12)) % 10
	var odds = gtin14[0] + gtin14[2] + gtin14[4] + gtin14[6] + gtin14[8] + gtin14[10] + gtin14[12]
	var even = gtin14[1] + gtin14[3] + gtin14[5] + gtin14[7] + gtin14[9] + gtin14[11]
	gtin14[digitCount] = ((-3 * odds) - even) % 10

	// This next line is because the MODULO function can return a negative number
	if gtin14[digitCount] < 0 {
		gtin14[digitCount] = gtin14[digitCount] + 10
	}

	return gtin14, nil
}
