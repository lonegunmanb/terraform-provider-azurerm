package pipelines

import (
	"encoding/json"
	"fmt"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

var _ CopySource = SapOdpSource{}

type SapOdpSource struct {
	AdditionalColumns *interface{} `json:"additionalColumns,omitempty"`
	ExtractionMode    *interface{} `json:"extractionMode,omitempty"`
	Projection        *interface{} `json:"projection,omitempty"`
	QueryTimeout      *interface{} `json:"queryTimeout,omitempty"`
	Selection         *interface{} `json:"selection,omitempty"`
	SubscriberProcess *interface{} `json:"subscriberProcess,omitempty"`

	// Fields inherited from CopySource

	DisableMetricsCollection *bool        `json:"disableMetricsCollection,omitempty"`
	MaxConcurrentConnections *int64       `json:"maxConcurrentConnections,omitempty"`
	SourceRetryCount         *int64       `json:"sourceRetryCount,omitempty"`
	SourceRetryWait          *interface{} `json:"sourceRetryWait,omitempty"`
	Type                     string       `json:"type"`
}

func (s SapOdpSource) CopySource() BaseCopySourceImpl {
	return BaseCopySourceImpl{
		DisableMetricsCollection: s.DisableMetricsCollection,
		MaxConcurrentConnections: s.MaxConcurrentConnections,
		SourceRetryCount:         s.SourceRetryCount,
		SourceRetryWait:          s.SourceRetryWait,
		Type:                     s.Type,
	}
}

var _ json.Marshaler = SapOdpSource{}

func (s SapOdpSource) MarshalJSON() ([]byte, error) {
	type wrapper SapOdpSource
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling SapOdpSource: %+v", err)
	}

	var decoded map[string]interface{}
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling SapOdpSource: %+v", err)
	}

	decoded["type"] = "SapOdpSource"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling SapOdpSource: %+v", err)
	}

	return encoded, nil
}
