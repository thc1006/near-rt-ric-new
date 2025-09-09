/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"time"
)

// OverallCompliance represents overall compliance status
type OverallCompliance struct {
	Compliant     bool                   `json:"compliant"`
	Score         float64                `json:"score"`
	LastEvaluated time.Time              `json:"lastEvaluated"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// TestSeverity and TestPriority definitions moved to types.go to avoid redeclarations