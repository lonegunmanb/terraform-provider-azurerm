// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package network

import (
	"fmt"
	"math/big"
)

// compareNumberOfIPAddresses compares two number_of_ip_addresses string values
// as arbitrary-precision integers. Returns a negative value if existing < expanded,
// zero if equal, and a positive value if existing > expanded.
// Uses math/big.Int to correctly handle IPv6-scale address counts that exceed int64.
func compareNumberOfIPAddresses(existing, expanded string) (int, error) {
	existingNum, ok := new(big.Int).SetString(existing, 10)
	if !ok || existingNum.Sign() < 0 {
		return 0, fmt.Errorf("parsing existing `number_of_ip_addresses` value %q as positive integer", existing)
	}
	expandedNum, ok := new(big.Int).SetString(expanded, 10)
	if !ok || expandedNum.Sign() < 0 {
		return 0, fmt.Errorf("parsing new `number_of_ip_addresses` value %q as positive integer", expanded)
	}
	return existingNum.Cmp(expandedNum), nil
}
