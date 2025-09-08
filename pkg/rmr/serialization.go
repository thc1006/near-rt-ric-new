package rmr

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
)

// Serializer provides methods for serializing and deserializing RMR messages
type Serializer struct{}

// SerializeJSON converts a struct to JSON bytes
func (s *Serializer) SerializeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// DeserializeJSON converts JSON bytes to a struct
func (s *Serializer) DeserializeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SerializeProtobuf converts a protobuf message to bytes
func (s *Serializer) SerializeProtobuf(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// DeserializeProtobuf converts bytes to a protobuf message
func (s *Serializer) DeserializeProtobuf(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// MessageWrapper provides a standard envelope for RMR messages
type MessageWrapper struct {
	Type     MessageType `json:"type"`
	Payload  []byte      `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Wrap creates a standard message envelope
func (s *Serializer) Wrap(msgType MessageType, payload []byte, metadata map[string]string) ([]byte, error) {
	wrapper := MessageWrapper{
		Type:     msgType,
		Payload:  payload,
		Metadata: metadata,
	}
	return s.SerializeJSON(wrapper)
}

// Unwrap extracts payload from a message envelope
func (s *Serializer) Unwrap(data []byte) (*MessageWrapper, error) {
	var wrapper MessageWrapper
	if err := s.DeserializeJSON(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unwrap message: %v", err)
	}
	return &wrapper, nil
}

// ValidateMessage checks message integrity and type
func (s *Serializer) ValidateMessage(data []byte) error {
	wrapper, err := s.Unwrap(data)
	if err != nil {
		return err
	}

	// Additional validation can be added here
	if len(wrapper.Payload) == 0 {
		return fmt.Errorf("empty message payload")
	}

	return nil
}