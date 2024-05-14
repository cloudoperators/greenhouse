// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package util

import "slices"

// AppendStringToSliceIfNotContains appends a string to a slice if it is not already present.
func AppendStringToSliceIfNotContains(theString string, theSlice []string) []string {
	if !slices.Contains(theSlice, theString) {
		theSlice = append(theSlice, theString)
	}
	return theSlice
}
