/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package utils

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

// Include returns true if the target string t is in the slice.
func Include(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

// Filter returns a list of strings with the paramater array filtered out
func Filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// RemoveDuplicates removes duplicates. Result is unordered
func RemoveDuplicates(vs []string) []string {
	dup := map[string]bool{}
	for e := range vs {
		dup[vs[e]] = true
	}

	var noDups []string
	for key := range dup {
		noDups = append(noDups, key)
	}
	return noDups
}
