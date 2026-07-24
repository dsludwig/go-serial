//
// Copyright 2014-2026 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package enumerator

import (
	"testing"
)

func parseAndReturnDeviceID(deviceID string) *PortDetails {
	res := &PortDetails{}
	parseDeviceID(deviceID, res)
	return res
}

func TestParseDeviceID(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		vid      string
		pid      string
		serialNo string
		channel  string
	}{
		{name: "FTDI FT232", deviceID: "FTDIBUS\\VID_0403+PID_6001+A6004CCFA\\0000", vid: "0403", pid: "6001", serialNo: "A6004CCFA"},
		{name: "Teensy USB serial", deviceID: "USB\\VID_16C0&PID_0483\\12345", vid: "16C0", pid: "0483", serialNo: "12345"},
		{name: "Arduino with serial number", deviceID: "USB\\VID_2341&PID_0000\\64936333936351400000", vid: "2341", pid: "0000", serialNo: "64936333936351400000"},
		{name: "Arduino with different serial number", deviceID: "USB\\VID_2341&PID_0000\\6493234373835191F1F1", vid: "2341", pid: "0000", serialNo: "6493234373835191F1F1"},
		{name: "Arduino MKR composite", deviceID: "USB\\VID_2341&PID_804E&MI_00\\6&279A3900&0&0000", vid: "2341", pid: "804E", serialNo: "", channel: "00"},
		{name: "Arduino MKR1000 bootloader", deviceID: "USB\\VID_2341&PID_004E\\5&C3DC240&0&1", vid: "2341", pid: "004E", serialNo: ""},
		{name: "Atmel EDBG debugger", deviceID: "USB\\VID_03EB&PID_2111&MI_01\\6&21F3553F&0&0001", vid: "03EB", pid: "2111", serialNo: "", channel: "01"},
		{name: "Arduino Zero composite", deviceID: "USB\\VID_2341&PID_804D&MI_00\\6&1026E213&0&0000", vid: "2341", pid: "804D", serialNo: "", channel: "00"},
		{name: "Arduino Zero bootloader", deviceID: "USB\\VID_2341&PID_004D\\5&C3DC240&0&1", vid: "2341", pid: "004D", serialNo: ""},
		{name: "Prolific PL2303", deviceID: "USB\\VID_067B&PID_2303\\6&2C4CB384&0&3", vid: "067B", pid: "2303", serialNo: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseAndReturnDeviceID(tt.deviceID)
			if !res.IsUSB {
				t.Fatal("expected IsUSB to be true")
			}
			if res.VID != tt.vid {
				t.Errorf("VID: got %q, expected %q", res.VID, tt.vid)
			}
			if res.PID != tt.pid {
				t.Errorf("PID: got %q, expected %q", res.PID, tt.pid)
			}
			if res.SerialNumber != tt.serialNo {
				t.Errorf("SerialNumber: got %q, expected %q", res.SerialNumber, tt.serialNo)
			}
			if res.UsbChannel != tt.channel {
				t.Errorf("UsbChannel: got %q, expected %q", res.UsbChannel, tt.channel)
			}
		})
	}
}

func TestApplyUsbAncestorInfo(t *testing.T) {
	tests := []struct {
		name       string
		deviceID   string
		ancestorID string
		done       bool
		serialNo   string
		channel    string
	}{
		{
			// The immediate parent of an FTDIBUS child is the composite USB
			// interface: no serial there, keep walking up.
			name:       "FT2232 channel A parent interface",
			deviceID:   "FTDIBUS\\VID_0403+PID_6010+FT891Q3A\\0000",
			ancestorID: "USB\\VID_0403&PID_6010&MI_00\\6&3B199AA1&0&0000",
			done:       false, serialNo: "FT891Q3A", channel: "",
		},
		{
			name:       "FT2232 channel A root device",
			deviceID:   "FTDIBUS\\VID_0403+PID_6010+FT891Q3A\\0000",
			ancestorID: "USB\\VID_0403&PID_6010\\FT891Q3",
			done:       true, serialNo: "FT891Q3", channel: "A",
		},
		{
			name:       "FT2232 channel B root device",
			deviceID:   "FTDIBUS\\VID_0403+PID_6010+FT891Q3B\\0000",
			ancestorID: "USB\\VID_0403&PID_6010\\FT891Q3",
			done:       true, serialNo: "FT891Q3", channel: "B",
		},
		{
			// EEPROM without a serial number: the root instance id is
			// windows-generated, no serial anywhere up the tree, and the
			// FTDIBUS-derived fragment is left as-is.
			name:       "FT2232 without EEPROM serial",
			deviceID:   "FTDIBUS\\VID_0403+PID_6010+6&3B199AA1&0&1A\\0000",
			ancestorID: "USB\\VID_0403&PID_6010\\6&3B199AA1&0",
			done:       false, serialNo: "6", channel: "",
		},
		{
			name:       "composite CDC root device",
			deviceID:   "USB\\VID_2341&PID_804E&MI_00\\6&279A3900&0&0000",
			ancestorID: "USB\\VID_2341&PID_804E\\95530343834351B0E1E1",
			done:       true, serialNo: "95530343834351B0E1E1", channel: "00",
		},
		{
			// A hub or controller above the device: different VID/PID stops
			// the search without touching the details.
			name:       "unrelated ancestor",
			deviceID:   "USB\\VID_2341&PID_804E&MI_00\\6&279A3900&0&0000",
			ancestorID: "USB\\VID_8086&PID_7AE0\\5&38E85D5&0&8",
			done:       true, serialNo: "", channel: "00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := parseAndReturnDeviceID(tt.deviceID)
			isFTDI := len(tt.deviceID) >= 7 && tt.deviceID[:7] == "FTDIBUS"
			done := applyUsbAncestorInfo(details, tt.ancestorID, details.SerialNumber, isFTDI)
			if done != tt.done {
				t.Errorf("done: got %v, expected %v", done, tt.done)
			}
			if details.SerialNumber != tt.serialNo {
				t.Errorf("SerialNumber: got %q, expected %q", details.SerialNumber, tt.serialNo)
			}
			if details.UsbChannel != tt.channel {
				t.Errorf("UsbChannel: got %q, expected %q", details.UsbChannel, tt.channel)
			}
		})
	}
}

func TestParseDeviceIDWithInvalidStrings(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
	}{
		{name: "unrecognized prefix", deviceID: "ABC"},
		{name: "USB prefix without fields", deviceID: "USB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseAndReturnDeviceID(tt.deviceID)
			if res.IsUSB {
				t.Fatal("expected IsUSB to be false")
			}
		})
	}
}
