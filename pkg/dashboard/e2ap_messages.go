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
	E2AP_PDU_INITIATING_MESSAGE   = 0
	E2AP_PDU_SUCCESSFUL_OUTCOME   = 1
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
	E2AP_CAUSE_RIC_REQUEST  = 1
	E2AP_CAUSE_RAN_FUNCTION = 2
	E2AP_CAUSE_E2_NODE      = 3
	E2AP_CAUSE_TRANSPORT    = 4
	E2AP_CAUSE_PROTOCOL     = 5
	E2AP_CAUSE_MISC         = 6
)


// E2SetupRequestMessage represents E2 Setup Request
type E2SetupRequestMessage struct {
	TransactionID                uint32                         `json:"transactionId"`
	GlobalE2NodeID               GlobalE2NodeID                 `json:"globalE2NodeId"`
	RANFunctions                 []RANFunctionItem              `json:"ranFunctions"`
	E2NodeComponentConfigAddList []E2NodeComponentConfigAddItem `json:"e2NodeComponentConfigAddList"`
}

// E2SetupResponseMessage represents E2 Setup Response
type E2SetupResponseMessage struct {
	TransactionID                uint32                     `json:"transactionId"`
	GlobalRICID                  GlobalRICID                `json:"globalRicId"`
	RANFunctionsAccepted         []RANFunctionIDItem        `json:"ranFunctionsAccepted"`
	RANFunctionsRejected         []RANFunctionIDCauseItem   `json:"ranFunctionsRejected"`
	E2NodeComponentConfigAckList []E2NodeComponentConfigAck `json:"e2NodeComponentConfigAckList"`
}

// E2SetupFailureMessage represents E2 Setup Failure
type E2SetupFailureMessage struct {
	TransactionID          uint32                  `json:"transactionId"`
	Cause                  E2APCause               `json:"cause"`
	TimeToWait             *uint32                 `json:"timeToWait,omitempty"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticalityDiagnostics,omitempty"`
}

// E2NodeConfigurationUpdateMessage represents E2 Node Configuration Update
type E2NodeConfigurationUpdateMessage struct {
	TransactionID                    uint32                             `json:"transactionId"`
	GlobalE2NodeID                   *GlobalE2NodeID                    `json:"globalE2NodeId,omitempty"`
	E2NodeComponentConfigAddList     []E2NodeComponentConfigAddItem     `json:"e2NodeComponentConfigAddList,omitempty"`
	E2NodeComponentConfigUpdateList  []E2NodeComponentConfigUpdateItem  `json:"e2NodeComponentConfigUpdateList,omitempty"`
	E2NodeComponentConfigRemovalList []E2NodeComponentConfigRemovalItem `json:"e2NodeComponentConfigRemovalList,omitempty"`
}

// E2ResetRequestMessage represents E2 Reset Request
type E2ResetRequestMessage struct {
	TransactionID uint32    `json:"transactionId"`
	Cause         E2APCause `json:"cause"`
}

// E2ResetResponseMessage represents E2 Reset Response
type E2ResetResponseMessage struct {
	TransactionID          uint32                  `json:"transactionId"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticalityDiagnostics,omitempty"`
}

// Supporting data structures


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
	E2NodeComponentInterfaceType   E2NodeComponentInterfaceType    `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID              E2NodeComponentID               `json:"e2NodeComponentId"`
	E2NodeComponentConfigAck       E2NodeComponentConfigAckType    `json:"e2NodeComponentConfigAck"`
	E2NodeComponentConfigUpdateAck *E2NodeComponentConfigUpdateAck `json:"e2NodeComponentConfigUpdateAck,omitempty"`
}

// E2NodeComponentInterfaceType represents interface type
type E2NodeComponentInterfaceType uint32

const (
	E2NodeComponentInterfaceTypeNG E2NodeComponentInterfaceType = 0
	E2NodeComponentInterfaceTypeXn E2NodeComponentInterfaceType = 1
	E2NodeComponentInterfaceTypeE1 E2NodeComponentInterfaceType = 2
	E2NodeComponentInterfaceTypeF1 E2NodeComponentInterfaceType = 3
	E2NodeComponentInterfaceTypeW1 E2NodeComponentInterfaceType = 4
	E2NodeComponentInterfaceTypeS1 E2NodeComponentInterfaceType = 5
	E2NodeComponentInterfaceTypeX2 E2NodeComponentInterfaceType = 6
)

// E2NodeComponentID type is now defined in types.go to avoid redeclaration

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
// E2NodeComponentConfigUpdateAck represents acknowledgment for configuration updatestype E2NodeComponentConfigUpdateAck struct {	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2NodeComponentInterfaceType"`	E2NodeComponentID            E2NodeComponentID            `json:"e2NodeComponentId"`	E2NodeComponentConfigAck     E2NodeComponentConfigAckType `json:"e2NodeComponentConfigAck"`}


// E2APCause represents E2AP cause
type E2APCause struct {
	CauseType  uint32 `json:"causeType"`
	CauseValue uint32 `json:"causeValue"`
}

// CriticalityDiagnostics represents criticality diagnostics
type CriticalityDiagnostics struct {
	ProcedureCode             *uint32                    `json:"procedureCode,omitempty"`
	TriggeringMessage         *uint32                    `json:"triggeringMessage,omitempty"`
	ProcedureCriticality      *uint32                    `json:"procedureCriticality,omitempty"`
	IEsCriticalityDiagnostics []IECriticalityDiagnostics `json:"iesCriticalityDiagnostics,omitempty"`
}

// IECriticalityDiagnostics represents IE criticality diagnostics
type IECriticalityDiagnostics struct {
	IEID          uint32 `json:"ieId"`
	IECriticality uint32 `json:"ieCriticality"`
	TypeOfError   uint32 `json:"typeOfError"`
}

// E2APEncoder provides ASN.1 PER encoding/decoding for E2AP messages
type E2APEncoder struct {
	// ASN.1 encoding context
	encodingBuffer []byte
	decodingBuffer []byte
}

// NewE2APEncoder creates a new E2AP encoder
func NewE2APEncoder() *E2APEncoder {
	return &E2APEncoder{
		encodingBuffer: make([]byte, 0, 4096),
		decodingBuffer: make([]byte, 0, 4096),
	}
}

// EncodeE2APMessage encodes an E2AP message to ASN.1 PER format
func (e *E2APEncoder) EncodeE2APMessage(msg *E2APMessage) ([]byte, error) {
	// This is a simplified ASN.1 PER encoding implementation
	// In a production system, this would use a proper ASN.1 library

	buf := make([]byte, 0, 1024)

	// Add message type (1 byte) - Map MessageType to PDU type
	var pduType uint8
	switch msg.MessageType {
	case E2APMessageTypeSetupRequest, E2APMessageTypeConfigurationUpdate:
		pduType = E2AP_PDU_INITIATING_MESSAGE
	case E2APMessageTypeSetupResponse, E2APMessageTypeConfigurationUpdateAck:
		pduType = E2AP_PDU_SUCCESSFUL_OUTCOME
	case E2APMessageTypeSetupFailure, E2APMessageTypeConfigurationUpdateFailure:
		pduType = E2AP_PDU_UNSUCCESSFUL_OUTCOME
	default:
		pduType = E2AP_PDU_INITIATING_MESSAGE
	}
	buf = append(buf, pduType)

	// Add procedure code (1 byte) - Map MessageType to procedure code
	var procedureCode uint8
	switch msg.MessageType {
	case E2APMessageTypeSetupRequest, E2APMessageTypeSetupResponse, E2APMessageTypeSetupFailure:
		procedureCode = E2AP_PROCEDURE_E2_SETUP
	case E2APMessageTypeConfigurationUpdate, E2APMessageTypeConfigurationUpdateAck, E2APMessageTypeConfigurationUpdateFailure:
		procedureCode = E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE
	default:
		procedureCode = E2AP_PROCEDURE_E2_SETUP
	}
	buf = append(buf, procedureCode)

	// Add criticality (1 byte) - Default to REJECT
	buf = append(buf, E2AP_CRITICALITY_REJECT)

	// Add message length placeholder (2 bytes)
	lengthPos := len(buf)
	buf = append(buf, 0, 0)

	// Encode message payload
	valueBytes := msg.Payload
	if valueBytes == nil {
		valueBytes = make([]byte, 0)
	}

	// Update length field
	binary.BigEndian.PutUint16(buf[lengthPos:], uint16(len(valueBytes)))

	// Append payload bytes
	buf = append(buf, valueBytes...)

	return buf, nil
}

// DecodeE2APMessage decodes an E2AP message from ASN.1 PER format
func (e *E2APEncoder) DecodeE2APMessage(data []byte) (*E2APMessage, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	msg := &E2APMessage{
		Payload:   data, // Store raw data in Payload field
		Timestamp: time.Now(),
	}

	// Parse header
	pduType := data[0]
	procedureCode := data[1]
	// criticality := data[2] // Not used in current E2APMessage struct

	// Map PDU type and procedure code to MessageType
	switch procedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		switch pduType {
		case E2AP_PDU_INITIATING_MESSAGE:
			msg.MessageType = E2APMessageTypeSetupRequest
		case E2AP_PDU_SUCCESSFUL_OUTCOME:
			msg.MessageType = E2APMessageTypeSetupResponse
		case E2AP_PDU_UNSUCCESSFUL_OUTCOME:
			msg.MessageType = E2APMessageTypeSetupFailure
		}
	case E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE:
		switch pduType {
		case E2AP_PDU_INITIATING_MESSAGE:
			msg.MessageType = E2APMessageTypeConfigurationUpdate
		case E2AP_PDU_SUCCESSFUL_OUTCOME:
			msg.MessageType = E2APMessageTypeConfigurationUpdateAck
		case E2AP_PDU_UNSUCCESSFUL_OUTCOME:
			msg.MessageType = E2APMessageTypeConfigurationUpdateFailure
		}
	default:
		msg.MessageType = E2APMessageTypeSetupRequest // Default
	}

	// Parse length
	length := binary.BigEndian.Uint16(data[3:5])

	if len(data) < int(5+length) {
		return nil, fmt.Errorf("incomplete message: expected %d bytes, got %d", 5+length, len(data))
	}

	// Extract message payload
	if length > 0 {
		msg.Payload = data[5 : 5+length]
	} else {
		msg.Payload = make([]byte, 0)
	}

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

// Enhanced ASN.1 PER encoding functions
func (e *E2APEncoder) encodeE2SetupRequest(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 1024)

	// Encode transaction ID (INTEGER 0..255)
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing or invalid transaction ID")
	}

	// Encode Global E2 Node ID
	if globalE2NodeID, ok := value["globalE2NodeId"].(GlobalE2NodeID); ok {
		nodeIDBytes, err := e.encodeGlobalE2NodeID(globalE2NodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to encode global E2 node ID: %w", err)
		}
		buf = append(buf, nodeIDBytes...)
	} else {
		return nil, fmt.Errorf("missing or invalid global E2 node ID")
	}

	// Encode RAN Functions List
	if ranFunctions, ok := value["ranFunctions"].([]RANFunctionItem); ok {
		ranFuncBytes, err := e.encodeRANFunctionsList(ranFunctions)
		if err != nil {
			return nil, fmt.Errorf("failed to encode RAN functions: %w", err)
		}
		buf = append(buf, ranFuncBytes...)
	}

	// Encode E2 Node Component Config Add List
	if componentConfig, ok := value["componentConfig"].([]E2NodeComponentConfigAddItem); ok {
		configBytes, err := e.encodeComponentConfigAddList(componentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to encode component config: %w", err)
		}
		buf = append(buf, configBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2SetupResponse(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 1024)

	// Encode transaction ID
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing transaction ID")
	}

	// Encode Global RIC ID
	if globalRICID, ok := value["globalRicId"].(GlobalRICID); ok {
		ricIDBytes, err := e.encodeGlobalRICID(globalRICID)
		if err != nil {
			return nil, fmt.Errorf("failed to encode global RIC ID: %w", err)
		}
		buf = append(buf, ricIDBytes...)
	}

	// Encode accepted RAN functions
	if accepted, ok := value["ranFunctionsAccepted"].([]RANFunctionIDItem); ok {
		acceptedBytes, err := e.encodeRANFunctionIDList(accepted)
		if err != nil {
			return nil, fmt.Errorf("failed to encode accepted RAN functions: %w", err)
		}
		buf = append(buf, acceptedBytes...)
	}

	// Encode rejected RAN functions
	if rejected, ok := value["ranFunctionsRejected"].([]RANFunctionIDCauseItem); ok {
		rejectedBytes, err := e.encodeRANFunctionIDCauseList(rejected)
		if err != nil {
			return nil, fmt.Errorf("failed to encode rejected RAN functions: %w", err)
		}
		buf = append(buf, rejectedBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2SetupFailure(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 512)

	// Encode transaction ID
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing transaction ID")
	}

	// Encode cause
	if cause, ok := value["cause"].(E2APCause); ok {
		causeBytes, err := e.encodeE2APCause(cause)
		if err != nil {
			return nil, fmt.Errorf("failed to encode cause: %w", err)
		}
		buf = append(buf, causeBytes...)
	} else {
		return nil, fmt.Errorf("missing cause")
	}

	// Encode optional time to wait
	if timeToWait, ok := value["timeToWait"].(*uint32); ok && timeToWait != nil {
		buf = append(buf, 1) // presence flag
		buf = append(buf, byte(*timeToWait))
	} else {
		buf = append(buf, 0) // not present
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2NodeConfigUpdate(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 1024)

	// Encode transaction ID
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing transaction ID")
	}

	// Encode optional Global E2 Node ID
	if globalE2NodeID, ok := value["globalE2NodeId"].(*GlobalE2NodeID); ok && globalE2NodeID != nil {
		buf = append(buf, 1) // presence flag
		nodeIDBytes, err := e.encodeGlobalE2NodeID(*globalE2NodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to encode global E2 node ID: %w", err)
		}
		buf = append(buf, nodeIDBytes...)
	} else {
		buf = append(buf, 0) // not present
	}

	// Encode config add list
	if configAddList, ok := value["configAddList"].([]E2NodeComponentConfigAddItem); ok {
		addListBytes, err := e.encodeComponentConfigAddList(configAddList)
		if err != nil {
			return nil, fmt.Errorf("failed to encode config add list: %w", err)
		}
		buf = append(buf, addListBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2ResetRequest(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Encode transaction ID
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing transaction ID")
	}

	// Encode cause
	if cause, ok := value["cause"].(E2APCause); ok {
		causeBytes, err := e.encodeE2APCause(cause)
		if err != nil {
			return nil, fmt.Errorf("failed to encode cause: %w", err)
		}
		buf = append(buf, causeBytes...)
	} else {
		return nil, fmt.Errorf("missing cause")
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2ResetResponse(value map[string]interface{}) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Encode transaction ID
	if transactionID, ok := value["transactionId"].(uint32); ok {
		buf = append(buf, byte(transactionID))
	} else {
		return nil, fmt.Errorf("missing transaction ID")
	}

	// Encode optional criticality diagnostics
	if critDiag, ok := value["criticalityDiagnostics"].(*CriticalityDiagnostics); ok && critDiag != nil {
		buf = append(buf, 1) // presence flag
		diagBytes, err := e.encodeCriticalityDiagnostics(*critDiag)
		if err != nil {
			return nil, fmt.Errorf("failed to encode criticality diagnostics: %w", err)
		}
		buf = append(buf, diagBytes...)
	} else {
		buf = append(buf, 0) // not present
	}

	return buf, nil
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

// Helper encoding functions for ASN.1 PER

func (e *E2APEncoder) encodeGlobalE2NodeID(nodeID GlobalE2NodeID) ([]byte, error) {
	buf := make([]byte, 0, 64)

	// Encode PLMN ID (3 bytes) - Use PLMNIdentity field
	plmnBytes := nodeID.PLMNIdentity
	if len(plmnBytes) != 3 {
		return nil, fmt.Errorf("invalid PLMN ID length: %d", len(plmnBytes))
	}
	buf = append(buf, plmnBytes...)

	// Encode Node ID (variable length) - Use E2NodeID field
	nodeIDBytes := nodeID.E2NodeID
	buf = append(buf, byte(len(nodeIDBytes))) // length
	buf = append(buf, nodeIDBytes...)

	return buf, nil
}

func (e *E2APEncoder) encodeGlobalRICID(ricID GlobalRICID) ([]byte, error) {
	buf := make([]byte, 0, 32)

	// Encode PLMN ID
	plmnBytes := ricID.PLMNIdentity
	if len(plmnBytes) != 3 {
		return nil, fmt.Errorf("invalid PLMN ID length: %d", len(plmnBytes))
	}
	buf = append(buf, plmnBytes...)

	// FIXED: Use correct field name RICId instead of RicID
	buf = append(buf, byte(len(ricID.RICId))) // length
	buf = append(buf, ricID.RICId...)

	return buf, nil
}

func (e *E2APEncoder) encodeRANFunctionsList(ranFunctions []RANFunctionItem) ([]byte, error) {
	buf := make([]byte, 0, 512)

	// Encode list length
	buf = append(buf, byte(len(ranFunctions)))

	for _, ranFunc := range ranFunctions {
		// Encode RAN Function ID (2 bytes)
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(ranFunc.RANFunctionID))
		buf = buf[:len(buf)+2]

		// Encode RAN Function Definition length and data
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(ranFunc.RANFunctionDefinition)))
		buf = buf[:len(buf)+2]
		buf = append(buf, ranFunc.RANFunctionDefinition...)

		// Encode RAN Function Revision (2 bytes)
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(ranFunc.RANFunctionRevision))
		buf = buf[:len(buf)+2]

		// Encode RAN Function OID
		oidBytes := []byte(ranFunc.RANFunctionOID)
		buf = append(buf, byte(len(oidBytes)))
		buf = append(buf, oidBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeRANFunctionIDList(ranFunctions []RANFunctionIDItem) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Encode list length
	buf = append(buf, byte(len(ranFunctions)))

	for _, ranFunc := range ranFunctions {
		// Encode RAN Function ID (2 bytes)
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(ranFunc.RANFunctionID))
		buf = buf[:len(buf)+2]

		// Encode RAN Function Revision (2 bytes)
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(ranFunc.RANFunctionRevision))
		buf = buf[:len(buf)+2]
	}

	return buf, nil
}

func (e *E2APEncoder) encodeRANFunctionIDCauseList(ranFunctions []RANFunctionIDCauseItem) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Encode list length
	buf = append(buf, byte(len(ranFunctions)))

	for _, ranFunc := range ranFunctions {
		// Encode RAN Function ID (2 bytes)
		binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(ranFunc.RANFunctionID))
		buf = buf[:len(buf)+2]

		// Encode Cause
		causeBytes, err := e.encodeE2APCause(ranFunc.Cause)
		if err != nil {
			return nil, fmt.Errorf("failed to encode cause for RAN function %d: %w", ranFunc.RANFunctionID, err)
		}
		buf = append(buf, causeBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeComponentConfigAddList(configList []E2NodeComponentConfigAddItem) ([]byte, error) {
	buf := make([]byte, 0, 512)

	// Encode list length
	buf = append(buf, byte(len(configList)))

	for _, config := range configList {
		// FIXED: Use string conversion instead of direct byte conversion
		buf = append(buf, byte(config.E2NodeComponentInterfaceType))

		// Encode component ID
		componentIDBytes, err := e.encodeE2NodeComponentID(config.E2NodeComponentID)
		if err != nil {
			return nil, fmt.Errorf("failed to encode component ID: %w", err)
		}
		buf = append(buf, componentIDBytes...)

		// Encode component configuration
		configBytes, err := e.encodeE2NodeComponentConfiguration(config.E2NodeComponentConfiguration)
		if err != nil {
			return nil, fmt.Errorf("failed to encode component configuration: %w", err)
		}
		buf = append(buf, configBytes...)
	}

	return buf, nil
}

func (e *E2APEncoder) encodeE2NodeComponentID(componentID E2NodeComponentID) ([]byte, error) {
	buf := make([]byte, 0, 64)

	// FIXED: Use Type field from types.go and convert correctly
	buf = append(buf, byte(len(string(componentID.Type)))) // length of type string
	buf = append(buf, []byte(string(componentID.Type))...)  // type as string bytes
	
	// Encode identifier
	idBytes := []byte(componentID.Identifier) // Use Identifier field from types.go
	buf = append(buf, byte(len(idBytes)))
	buf = append(buf, idBytes...)

	return buf, nil
}

func (e *E2APEncoder) encodeE2NodeComponentConfiguration(config E2NodeComponentConfiguration) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Encode request part length and data
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(config.E2NodeComponentRequestPart)))
	buf = buf[:len(buf)+2]
	buf = append(buf, config.E2NodeComponentRequestPart...)

	// Encode response part length and data
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(config.E2NodeComponentResponsePart)))
	buf = buf[:len(buf)+2]
	buf = append(buf, config.E2NodeComponentResponsePart...)

	return buf, nil
}

func (e *E2APEncoder) encodeE2APCause(cause E2APCause) ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = byte(cause.CauseType)
	buf[1] = byte(cause.CauseValue)
	return buf, nil
}

func (e *E2APEncoder) encodeCriticalityDiagnostics(diag CriticalityDiagnostics) ([]byte, error) {
	buf := make([]byte, 0, 128)

	// Encode optional procedure code
	if diag.ProcedureCode != nil {
		buf = append(buf, 1) // presence flag
		buf = append(buf, byte(*diag.ProcedureCode))
	} else {
		buf = append(buf, 0) // not present
	}

	// Encode optional triggering message
	if diag.TriggeringMessage != nil {
		buf = append(buf, 1) // presence flag
		buf = append(buf, byte(*diag.TriggeringMessage))
	} else {
		buf = append(buf, 0) // not present
	}

	// Encode optional procedure criticality
	if diag.ProcedureCriticality != nil {
		buf = append(buf, 1) // presence flag
		buf = append(buf, byte(*diag.ProcedureCriticality))
	} else {
		buf = append(buf, 0) // not present
	}

	// Encode IEs criticality diagnostics list
	buf = append(buf, byte(len(diag.IEsCriticalityDiagnostics)))
	for _, ieDiag := range diag.IEsCriticalityDiagnostics {
		binary.BigEndian.PutUint32(buf[len(buf):len(buf)+4], ieDiag.IEID)
		buf = buf[:len(buf)+4]
		buf = append(buf, byte(ieDiag.IECriticality))
		buf = append(buf, byte(ieDiag.TypeOfError))
	}

	return buf, nil
}

// Helper function to map node types (commented out since E2NodeType is not defined)
// func (e *E2APEncoder) nodeTypeToInt(nodeType E2NodeType) int {
// 	switch nodeType {
// 	case E2NodeTypeGNB:
// 		return 0
// 	case E2NodeTypeENB:
// 		return 1
// 	case E2NodeTypeOCU:
// 		return 2
// 	case E2NodeTypeODU:
// 		return 3
// 	case E2NodeTypeOCUCP:
// 		return 4
// 	case E2NodeTypeOCUUP:
// 		return 5
// 	default:
// 		return 0
// 	}
// }