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

	// Encode PLMN ID (3 bytes)
	if len(nodeID.PlmnID) != 6 { // Hex string representation
		return nil, fmt.Errorf("invalid PLMN ID length: %d", len(nodeID.PlmnID))
	}

	plmnBytes := make([]byte, 3)
	for i := 0; i < 3; i++ {
		if _, err := fmt.Sscanf(nodeID.PlmnID[i*2:i*2+2], "%02x", &plmnBytes[i]); err != nil {
			return nil, fmt.Errorf("invalid PLMN ID format: %w", err)
		}
	}
	buf = append(buf, plmnBytes...)

	// Encode Node ID (variable length)
	nodeIDBytes := []byte(nodeID.NodeID)
	buf = append(buf, byte(len(nodeIDBytes))) // length
	buf = append(buf, nodeIDBytes...)

	// Encode Node Type
	buf = append(buf, byte(e.nodeTypeToInt(nodeID.Type)))

	return buf, nil
}

func (e *E2APEncoder) encodeGlobalRICID(ricID GlobalRICID) ([]byte, error) {
	buf := make([]byte, 0, 32)

	// Encode PLMN ID
	if len(ricID.PlmnID) != 6 {
		return nil, fmt.Errorf("invalid PLMN ID length: %d", len(ricID.PlmnID))
	}

	plmnBytes := make([]byte, 3)
	for i := 0; i < 3; i++ {
		if _, err := fmt.Sscanf(ricID.PlmnID[i*2:i*2+2], "%02x", &plmnBytes[i]); err != nil {
			return nil, fmt.Errorf("invalid PLMN ID format: %w", err)
		}
	}
	buf = append(buf, plmnBytes...)

	// Encode RIC ID
	buf = append(buf, byte(len(ricID.RicID))) // length
	buf = append(buf, ricID.RicID...)

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
		// Encode interface type
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

	// Determine which component type is present and encode accordingly
	if componentID.E2NodeComponentTypeNG != nil {
		buf = append(buf, 0) // NG type indicator
		buf = append(buf, byte(len(componentID.E2NodeComponentTypeNG.AMFID)))
		buf = append(buf, componentID.E2NodeComponentTypeNG.AMFID...)
	} else if componentID.E2NodeComponentTypeXn != nil {
		buf = append(buf, 1) // Xn type indicator
		buf = append(buf, byte(len(componentID.E2NodeComponentTypeXn.GlobalNGRANNodeID)))
		buf = append(buf, componentID.E2NodeComponentTypeXn.GlobalNGRANNodeID...)
	} else if componentID.E2NodeComponentTypeE1 != nil {
		buf = append(buf, 2) // E1 type indicator
		binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], componentID.E2NodeComponentTypeE1.GNBCUCPID)
		buf = buf[:len(buf)+8]
	} else if componentID.E2NodeComponentTypeF1 != nil {
		buf = append(buf, 3) // F1 type indicator
		binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], componentID.E2NodeComponentTypeF1.GNBDUID)
		buf = buf[:len(buf)+8]
	} else if componentID.E2NodeComponentTypeW1 != nil {
		buf = append(buf, 4) // W1 type indicator
		binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], componentID.E2NodeComponentTypeW1.NGENBDUID)
		buf = buf[:len(buf)+8]
	} else if componentID.E2NodeComponentTypeS1 != nil {
		buf = append(buf, 5) // S1 type indicator
		buf = append(buf, byte(len(componentID.E2NodeComponentTypeS1.MMEID)))
		buf = append(buf, componentID.E2NodeComponentTypeS1.MMEID...)
	} else if componentID.E2NodeComponentTypeX2 != nil {
		buf = append(buf, 6) // X2 type indicator
		buf = append(buf, byte(len(componentID.E2NodeComponentTypeX2.GlobalENBID)))
		buf = append(buf, componentID.E2NodeComponentTypeX2.GlobalENBID...)
	} else {
		return nil, fmt.Errorf("no component type specified")
	}

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

func (e *E2APEncoder) nodeTypeToInt(nodeType E2NodeType) int {
	switch nodeType {
	case E2NodeTypeGNB:
		return 0
	case E2NodeTypeENB:
		return 1
	case E2NodeTypeOCU:
		return 2
	case E2NodeTypeODU:
		return 3
	case E2NodeTypeOCUCP:
		return 4
	case E2NodeTypeOCUUP:
		return 5
	default:
		return 0
	}
}
