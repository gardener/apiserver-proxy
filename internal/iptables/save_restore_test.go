/*
SPDX-FileCopyrightText: 2018 The Kubernetes Authors.
SPDX-License-Identifier: Apache-2.0
*/

package iptables

import (
	"testing"
)

func TestReadLinesFromByteBuffer(t *testing.T) {
	testFn := func(byteArray []byte, expected []string) {
		index := 0
		readIndex := 0
		for ; readIndex < len(byteArray); index++ {
			line, n := readLine(readIndex, byteArray)
			readIndex = n
			if expected[index] != string(line) {
				t.Errorf("expected:%q, actual:%q", expected[index], line)
			}
		} // for
		if readIndex < len(byteArray) {
			t.Errorf("Byte buffer was only partially read. Buffer length is:%d, readIndex is:%d", len(byteArray), readIndex)
		}
		if index < len(expected) {
			t.Errorf("All expected strings were not compared. expected arr length:%d, matched count:%d", len(expected), index-1)
		}
	}

	byteArray1 := []byte("\n  Line 1  \n\n\n L ine4  \nLine 5 \n \n")
	expected1 := []string{"", "Line 1", "", "", "L ine4", "Line 5", ""}
	testFn(byteArray1, expected1)

	byteArray1 = []byte("")
	expected1 = []string{}
	testFn(byteArray1, expected1)

	byteArray1 = []byte("\n\n")
	expected1 = []string{"", ""}
	testFn(byteArray1, expected1)
}
