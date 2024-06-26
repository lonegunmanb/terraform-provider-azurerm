package firewallpolicies

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type FirewallPolicyHTTPHeaderToInsert struct {
	HeaderName  *string `json:"headerName,omitempty"`
	HeaderValue *string `json:"headerValue,omitempty"`
}
