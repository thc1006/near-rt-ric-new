/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/binary"
	"fmt"
	"time"
)

// E2AP Message Types per O-RAN.WG3.E2AP-R003
const (
	E2AP_PDU_INITIATING_MESSAGE = 0
	E2AP_PDU_SUCCESSFUL_OUTCOME  = 1
	E2AP_PDU_UNSUCCESSFUL_OUTCOME = 2
)

// E2AP Procedure Codes
const (
	E2AP_PROCEDURE_E2_SETUP                = 1
	E2AP_PROCEDURE_ERROR_INDICATION        = 2
	E2AP_PROCEDURE_RESET                   = 3
	E2AP_PROCEDURE_RIC_CONTROL             = 4
	E2AP_PROCEDURE_RIC_INDICATION          = 5
	E2AP_PROCEDURE_RIC_SERVICE_QUERY       = 6
	E2AP_PROCEDURE_RIC_SERVICE_UPDATE      = 7
	E2AP_PROCEDURE_RIC_SUBSCRIPTION        = 8
	E2AP_PROCEDURE_RIC_SUBSCRIPTION_DELETE = 9
	E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE   = 10
	E2AP_PROCEDURE_E2_CONNECTION_UPDATE    = 11
)

// E2AP Criticality values
const (
	E2AP_CRITICALITY_REJECT = 0
	E2AP_CRITICALITY_IGNORE = 1
	E2AP_CRITICALITY_NOTIFY = 2
)

// E2AP Cause values
const (
	E2AP_CAUSE_RIC_REQUEST = 1
	E2AP_CAUSE_RAN_FUNCTION = 2
	E2AP_CAUSE_E2_NODE = 3
	E2AP_CAUSE_TRANSPORT = 4
	E2AP_CAUSE_PROTOCOL = 5
	E2AP_CAUSE_MISC = 6
)

// E2APMessage represents a generic E2AP message
type E2APMessage struct {
	PDUType       uint8                  `json:"pduType"`
	ProcedureCode uint8                  `json:"procedureCode"`
	Criticality   uint8                  `json:"criticality"`
	Value         map[string]interface{} `json:"value"`
	RawData       []byte                 `json:"rawData,omitempty"`
}

// E2SetupRequestMessage represents E2 Setup Request
type E2SetupRequestMessage struct {
	TransactionID   uint32                `json:"transactionId"`
	GlobalE2NodeID  GlobalE2NodeID        `json:"globalE2NodeId"`
	RANFunctions    []RANFunctionItem     `json:"ranFunctions"`
	E2NodeComponentConfigAddList []E2NodeComponentConfigAddItem `json:"e2NodeComponentConfigAddList"`
}

// E2SetupResponseMessage represents E2 Setup Response
type E2SetupResponseMessage struct {
	TransactionID            uint32                    `json:"transactionId"`
	GlobalRICID              GlobalRICID               `json:"globalRicId"`
	RANFunctionsAccepted     []RANFunctionIDItem       `json:"ranFunctionsAccepted"`
	RANFunctionsRejected     []RANFunctionIDCauseItem  `json:"ranFunctionsRejected"`
	E2NodeComponentConfigAckList []E2NodeComponentConfigAck `json:"e2NodeComponentConfigAckList"`
}

// E2SetupFailureMessage represents E2 Setup Failure
type E2SetupFailureMessage struct {
	TransactionID uint32    `json:"transactionId"`
	Cause         E2APCause `json:"cause"`
	TimeToWait    *uint32   `json:"timeToWait,omitempty"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticalityDiagnostics,omitempty"`
}

// E2NodeConfigurationUpdateMessage represents E2 Node Configuration Update
type E2NodeConfigurationUpdateMessage struct {
	TransactionID                        uint32                              `json:"transactionId"`
	GlobalE2NodeID                       *GlobalE2NodeID                     `json:"globalE2NodeId,omitempty"`
	E2NodeComponentConfigAddList         []E2NodeComponentConfigAddItem     `json:"e2NodeComponentConfigAddList,omitempty"`
	E2NodeComponentConfigUpdateList      []E2NodeComponentConfigUpdateItem  `json:"e2NodeComponentConfigUpdateList,omitempty"`
	E2NodeComponentConfigRemovalList     []E2NodeComponentConfigRemovalItem `json:"e2NodeComponentConfigRemovalList,omitempty"`
}

// E2ResetRequestMessage represents E2 Reset Request
type E2ResetRequestMessage struct {
	TransactionID uint32    `json:"transactionId"`
	Cause         E2APCause `json:"cause"`
}

// E2ResetResponseMessage represents E2 Reset Response
type E2ResetResponseMessage struct {
	TransactionID uint32                    `json:"transactionId"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticalityDiagnostics,omitempty"`
}

// Supporting data structures

// GlobalRICID represents Global RIC ID
type GlobalRICID struct {
	PlmnID string `json:"plmnId"`
	RicID  []byte `json:"ricId"`
}

// RANFunctionItem represents RAN Function Item
type RANFunctionItem struct {
	RANFunctionID         uint32 `json:"ranFunctionId"`
	RANFunctionDefinition []byte `json:"ranFunctionDefinition"`
	RANFunctionRevision   uint32 `json:"ranFunctionRevision"`
	RANFunctionOID        string `json:"ranFunctionOid"`
}

// RANFunctionIDItem represents RAN Function ID Item
type RANFunctionIDItem struct {
	RANFunctionID       uint32 `json:"ranFunctionId"`
	RANFunctionRevision uint32 `json:"ranFunctionRevision"`
}

// RANFunctionIDCauseItem represents RAN Function ID Cause Item
type RANFunctionIDCauseItem struct {
	RANFunctionID uint32    `json:"ranFunctionId"`
	Cause         E2APCause `json:"cause"`
}

// E2NodeComponentConfigAddItem represents E2 Node Component Config Add Item
type E2NodeComponentConfigAddItem struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            E2NodeComponentID            `json:"e2NodeComponentId"`
	E2NodeComponentConfiguration E2NodeComponentConfiguration `json:"e2NodeComponentConfiguration"`
}

// E2NodeComponentConfigUpdateItem represents E2 Node Component Config Update Item
type E2NodeComponentConfigUpdateItem struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            E2NodeComponentID            `json:"e2NodeComponentId"`
	E2NodeComponentConfiguration E2NodeComponentConfiguration `json:"e2NodeComponentConfiguration"`
}

// E2NodeComponentConfigRemovalItem represents E2 Node Component Config Removal Item
type E2NodeComponentConfigRemovalItem struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            E2NodeComponentID            `json:"e2NodeComponentId"`
}

// E2NodeComponentConfigAck represents E2 Node Component Config Ack
type E2NodeComponentConfigAck struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            E2NodeComponentID            `json:"e2NodeComponentId"`
	E2NodeComponentConfigAck     E2NodeComponentConfigAckType `json:"e2NodeComponentConfigAck"`
	E2NodeComponentConfigUpdateAck *E2NodeComponentConfigUpdateAck `json:"e2NodeComponentConfigUpdateAck,omitempty"`
}

// E2NodeComponentInterfaceType represents interface type
type E2NodeComponentInterfaceType uint32

const (
	E2NodeComponentInterfaceTypeNG  E2NodeComponentInterfaceType = 0
	E2NodeComponentInterfaceTypeXn  E2NodeComponentInterfaceType = 1
	E2NodeComponentInterfaceTypeE1  E2NodeComponentInterfaceType = 2
	E2NodeComponentInterfaceTypeF1  E2NodeComponentInterfaceType = 3
	E2NodeComponentInterfaceTypeW1  E2NodeComponentInterfaceType = 4
	E2NodeComponentInterfaceTypeS1  E2NodeComponentInterfaceType = 5
	E2NodeComponentInterfaceTypeX2  E2NodeComponentInterfaceType = 6
)

// E2NodeComponentID represents component ID
type E2NodeComponentID struct {
	E2NodeComponentTypeNG  *E2NodeComponentIDNG  `json:"e2NodeComponentTypeNg,omitempty"`
	E2NodeComponentTypeXn  *E2NodeComponentIDXn  `json:"e2NodeComponentTypeXn,omitempty"`
	E2NodeComponentTypeE1  *E2NodeComponentIDE1  `json:"e2NodeComponentTypeE1,omitempty"`
	E2NodeComponentTypeF1  *E2NodeComponentIDF1  `json:"e2NodeComponentTypeF1,omitempty"`
	E2NodeComponentTypeW1  *E2NodeComponentIDW1  `json:"e2NodeComponentTypeW1,omitempty"`
	E2NodeComponentTypeS1  *E2NodeComponentIDS1  `json:"e2NodeComponentTypeS1,omitempty"`
	E2NodeComponentTypeX2  *E2NodeComponentIDX2  `json:"e2NodeComponentTypeX2,omitempty"`
}

// Component ID types
type E2NodeComponentIDNG struct {
	AMFID []byte `json:"amfId"`
}

type E2NodeComponentIDXn struct {
	GlobalNGRANNodeID []byte `json:"globalNgRanNodeId"`
}

type E2NodeComponentIDE1 struct {
	GNBCUCPID uint64 `json:"gnbCuCpId"`
}

type E2NodeComponentIDF1 struct {
	GNBDUID uint64 `json:"gnbDuId"`
}

type E2NodeComponentIDW1 struct {
	NGENBDUID uint64 `json:"ngEnbDuId"`
}

type E2NodeComponentIDS1 struct {
	MMEID []byte `json:"mmeId"`
}

type E2NodeComponentIDX2 struct {
	GlobalENBID []byte `json:"globalEnbId"`
}

// E2NodeComponentConfiguration represents component configuration
type E2NodeComponentConfiguration struct {
	E2NodeComponentRequestPart  []byte `json:"e2NodeComponentRequestPart"`
	E2NodeComponentResponsePart []byte `json:"e2NodeComponentResponsePart"`
}

// E2NodeComponentConfigAckType represents config ack type
type E2NodeComponentConfigAckType uint32

const (
	E2NodeComponentConfigAckTypeSuccess E2NodeComponentConfigAckType = 0
	E2NodeComponentConfigAckTypeFailure E2NodeComponentConfigAckType = 1
)

// E2NodeComponentConfigUpdateAck represents config update ack
type E2NodeComponentConfigUpdateAck struct {
	UpdateOutcome E2NodeComponentConfigAckType `json:"updateOutcome"`
	FailureCause  *E2APCause                   `json:"failureCause,omitempty"`
}

// E2APCause represents E2AP cause
type E2APCause struct {
	CauseType uint32 `json:"causeType"`
	CauseValue uint32 `json:"causeValue"`
}

// CriticalityDiagnostics represents criticality diagnostics
type CriticalityDiagnostics struct {
	ProcedureCode         *uint32                           `json:"procedureCode,omitempty"`
	TriggeringMessage     *uint32                           `json:"triggeringMessage,omitempty"`
	ProcedureCriticality  *uint32                           `json:"procedureCriticality,omitempty"`
	IEsCriticalityDiagnostics []IECriticalityDiagnostics    `json:"iesCriticalityDiagnostics,omitempty"`
}

// IECriticalityDiagnostics represents IE criticality diagnostics
type IECriticalityDiagnostics struct {
	IEID         uint32 `json:"ieId"`
	IECriticality uint32 `json:"ieCriticality"`
	TypeOfError  uint32 `json:"typeOfError"`
}

// E2APEncoder provides ASN.1 PER encoding/decoding for E2AP messages
type E2APEncoder struct{}

// NewE2APEncoder creates a new E2AP encoder
func NewE2APEncoder() *E2APEncoder {
	return &E2APEncoder{}
}

// EncodeE2APMessage encodes an E2AP message to ASN.1 PER format
func (e *E2APEncoder) EncodeE2APMessage(msg *E2APMessage) ([]byte, error) {
	// This is a simplified ASN.1 PER encoding implementation
	// In a production system, this would use a proper ASN.1 library
	
	buf := make([]byte, 0, 1024)
	
	// Add PDU type (1 byte)
	buf = append(buf, msg.PDUType)
	
	// Add procedure code (1 byte)
	buf = append(buf, msg.ProcedureCode)
	
	// Add criticality (1 byte)
	buf = append(buf, msg.Criticality)
	
	// Add message length placeholder (2 bytes)
	lengthPos := len(buf)
	buf = append(buf, 0, 0)
	
	// Encode message value based on procedure code
	valueBytes, err := e.encodeMessageValue(msg.ProcedureCode, msg.PDUType, msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message value: %w", err)
	}
	
	// Update length field
	binary.BigEndian.PutUint16(buf[lengthPos:], uint16(len(valueBytes)))
	
	// Append value bytes
	buf = append(buf, valueBytes...)
	
	return buf, nil
}

// DecodeE2APMessage decodes an E2AP message from ASN.1 PER format
func (e *E2APEncoder) DecodeE2APMessage(data []byte) (*E2APMessage, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}
	
	msg := &E2APMessage{
		RawData: data,
	}
	
	// Parse header
	msg.PDUType = data[0]
	msg.ProcedureCode = data[1]
	msg.Criticality = data[2]
	
	// Parse length
	length := binary.BigEndian.Uint16(data[3:5])
	
	if len(data) < int(5+length) {
		return nil, fmt.Errorf("incomplete message: expected %d bytes, got %d", 5+length, len(data))
	}
	
	// Decode message value
	valueData := data[5 : 5+length]
	value, err := e.decodeMessageValue(msg.ProcedureCode, msg.PDUType, valueData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message value: %w", err)
	}
	
	msg.Value = value
	return msg, nil
}

// encodeMessageValue encodes message value based on procedure code and PDU type
func (e *E2APEncoder) encodeMessageValue(procedureCode, pduType uint8, value map[string]interface{}) ([]byte, error) {
	// Simplified encoding - in production, use proper ASN.1 library
	buf := make([]byte, 0, 512)
	
	switch procedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.encodeE2SetupRequest(value)
		} else if pduType == E2AP_PDU_SUCCESSFUL_OUTCOME {
			return e.encodeE2SetupResponse(value)
		} else if pduType == E2AP_PDU_UNSUCCESSFUL_OUTCOME {
			return e.encodeE2SetupFailure(value)
		}
	case E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.encodeE2NodeConfigUpdate(value)
		}
	case E2AP_PROCEDURE_RESET:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.encodeE2ResetRequest(value)
		} else if pduType == E2AP_PDU_SUCCESSFUL_OUTCOME {
			return e.encodeE2ResetResponse(value)
		}
	}
	
	return buf, nil
}

// decodeMessageValue decodes message value based on procedure code and PDU type
func (e *E2APEncoder) decodeMessageValue(procedureCode, pduType uint8, data []byte) (map[string]interface{}, error) {
	value := make(map[string]interface{})
	
	switch procedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.decodeE2SetupRequest(data)
		} else if pduType == E2AP_PDU_SUCCESSFUL_OUTCOME {
			return e.decodeE2SetupResponse(data)
		} else if pduType == E2AP_PDU_UNSUCCESSFUL_OUTCOME {
			return e.decodeE2SetupFailure(data)
		}
	case E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.decodeE2NodeConfigUpdate(data)
		}
	case E2AP_PROCEDURE_RESET:
		if pduType == E2AP_PDU_INITIATING_MESSAGE {
			return e.decodeE2ResetRequest(data)
		} else if pduType == E2AP_PDU_SUCCESSFUL_OUTCOME {
			return e.decodeE2ResetResponse(data)
		}
	}
	
	return value, nil
}

// Simplified encoding functions (in production, use proper ASN.1 library)
func (e *E2APEncoder) encodeE2SetupRequest(value map[string]interface{}) ([]byte, error) {
	// Simplified implementation
	return []byte("E2SetupRequest"), nil
}

func (e *E2APEncoder) encodeE2SetupResponse(value map[string]interface{}) ([]byte, error) {
	return []byte("E2SetupResponse"), nil
}

func (e *E2APEncoder) encodeE2SetupFailure(value map[string]interface{}) ([]byte, error) {
	return []byte("E2SetupFailure"), nil
}

func (e *E2APEncoder) encodeE2NodeConfigUpdate(value map[string]interface{}) ([]byte, error) {
	return []byte("E2NodeConfigUpdate"), nil
}

func (e *E2APEncoder) encodeE2ResetRequest(value map[string]interface{}) ([]byte, error) {
	return []byte("E2ResetRequest"), nil
}

func (e *E2APEncoder) encodeE2ResetResponse(value map[string]interface{}) ([]byte, error) {
	return []byte("E2ResetResponse"), nil
}

// Simplified decoding functions
func (e *E2APEncoder) decodeE2SetupRequest(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2SetupRequest",
		"timestamp":   time.Now(),
	}, nil
}

func (e *E2APEncoder) decodeE2SetupResponse(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2SetupResponse",
		"timestamp":   time.Now(),
	}, nil
}

func (e *E2APEncoder) decodeE2SetupFailure(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2SetupFailure",
		"timestamp":   time.Now(),
	}, nil
}

func (e *E2APEncoder) decodeE2NodeConfigUpdate(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2NodeConfigUpdate",
		"timestamp":   time.Now(),
	}, nil
}

func (e *E2APEncoder) decodeE2ResetRequest(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2ResetRequest",
		"timestamp":   time.Now(),
	}, nil
}

func (e *E2APEncoder) decodeE2ResetResponse(data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{
		"messageType": "E2ResetResponse",
		"timestamp":   time.Now(),
	}, nil
}