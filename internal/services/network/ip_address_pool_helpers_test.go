// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package network

import (
	"testing"
)

func TestCompareNumberOfIPAddresses(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		expanded string
		wantCmp  int // -1 = existing < expanded, 0 = equal, 1 = existing > expanded
		wantErr  bool
	}{
		{
			name:     "increase from 32 to 128 should be allowed (regression case from issue #32258)",
			existing: "32",
			expanded: "128",
			wantCmp:  -1,
		},
		{
			name:     "decrease from 128 to 32 should be blocked",
			existing: "128",
			expanded: "32",
			wantCmp:  1,
		},
		{
			name:     "increase from 10 to 50 should be allowed",
			existing: "10",
			expanded: "50",
			wantCmp:  -1,
		},
		{
			name:     "decrease from 50 to 10 should be blocked",
			existing: "50",
			expanded: "10",
			wantCmp:  1,
		},
		{
			name:     "same value should be allowed",
			existing: "256",
			expanded: "256",
			wantCmp:  0,
		},
		{
			name:     "increase from 9 to 10 should be allowed (single to double digit)",
			existing: "9",
			expanded: "10",
			wantCmp:  -1,
		},
		{
			name:     "IPv6 scale increase should be allowed",
			existing: "18446744073709551616",
			expanded: "5192296858534827628530496329220096",
			wantCmp:  -1,
		},
		{
			name:     "IPv6 scale decrease should be blocked",
			existing: "5192296858534827628530496329220096",
			expanded: "18446744073709551616",
			wantCmp:  1,
		},
		{
			name:     "invalid existing value returns error",
			existing: "abc",
			expanded: "128",
			wantErr:  true,
		},
		{
			name:     "invalid expanded value returns error",
			existing: "32",
			expanded: "xyz",
			wantErr:  true,
		},
		{
			name:     "zero existing value compared to positive",
			existing: "0",
			expanded: "10",
			wantCmp:  -1,
		},
		{
			name:     "positive compared to zero expanded value",
			existing: "10",
			expanded: "0",
			wantCmp:  1,
		},
		{
			name:     "negative existing value returns error",
			existing: "-128",
			expanded: "128",
			wantErr:  true,
		},
		{
			name:     "negative expanded value returns error",
			existing: "32",
			expanded: "-64",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareNumberOfIPAddresses(tt.existing, tt.expanded)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Normalize to -1, 0, 1 for comparison
			gotSign := 0
			if got > 0 {
				gotSign = 1
			} else if got < 0 {
				gotSign = -1
			}
			if gotSign != tt.wantCmp {
				t.Errorf("compareNumberOfIPAddresses(%q, %q) = %d (sign %d), want sign %d",
					tt.existing, tt.expanded, got, gotSign, tt.wantCmp)
			}
		})
	}
}
