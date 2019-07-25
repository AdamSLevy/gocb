package gocb

import (
	"encoding/json"

	gocbcore "github.com/couchbase/gocbcore/v8"
)

// Transcoder provides an interface for transforming Go values to and
// from raw bytes for storage and retreival from Couchbase data storage.
type Transcoder interface {
	// Decodes retrieved bytes into a Go type.
	Decode([]byte, uint32, interface{}) error

	// Encodes a Go type into bytes for storage.
	Encode(interface{}) ([]byte, uint32, error)
}

// JSONSerializer is used a Transcoder for serialization/deserialization of JSON datatype values.
type JSONSerializer interface {
	// Serialize serializes an interface into bytes.
	Serialize(value interface{}) ([]byte, error)

	// Deserialize deserializes json bytes into an interface.
	Deserialize(bytes []byte, out interface{}) error
}

// DefaultTranscoder implements the default transcoding behaviour of
// all Couchbase SDKs.
type DefaultTranscoder struct {
	serializer JSONSerializer
}

// NewDefaultTranscoder returns a new DefaultTranscoder initialized to use DefaultSerializer.∂
func NewDefaultTranscoder(serializer JSONSerializer) *DefaultTranscoder {
	return &DefaultTranscoder{
		serializer: &DefaultJSONSerializer{},
	}
}

// Decode applies the default Couchbase transcoding behaviour to decode into a Go type.
func (t *DefaultTranscoder) Decode(bytes []byte, flags uint32, out interface{}) error {
	valueType, compression := gocbcore.DecodeCommonFlags(flags)

	// Make sure compression is disabled
	if compression != gocbcore.NoCompression {
		return clientError{"Unexpected value compression"}
	}

	// Normal types of decoding
	if valueType == gocbcore.BinaryType {
		switch typedOut := out.(type) {
		case *[]byte:
			*typedOut = bytes
			return nil
		case *interface{}:
			*typedOut = bytes
			return nil
		default:
			return clientError{"You must encode binary in a byte array or interface"}
		}
	} else if valueType == gocbcore.StringType {
		switch typedOut := out.(type) {
		case *string:
			*typedOut = string(bytes)
			return nil
		case *interface{}:
			*typedOut = string(bytes)
			return nil
		default:
			return clientError{"You must encode a string in a string or interface"}
		}
	} else if valueType == gocbcore.JsonType {
		err := t.serializer.Deserialize(bytes, &out)
		if err != nil {
			return err
		}
		return nil
	}

	return clientError{"Unexpected flags value"}
}

// Encode applies the default Couchbase transcoding behaviour to encode a Go type.
func (t *DefaultTranscoder) Encode(value interface{}) ([]byte, uint32, error) {
	var bytes []byte
	var flags uint32
	var err error

	switch typeValue := value.(type) {
	case []byte:
		bytes = typeValue
		flags = gocbcore.EncodeCommonFlags(gocbcore.BinaryType, gocbcore.NoCompression)
	case *[]byte:
		bytes = *typeValue
		flags = gocbcore.EncodeCommonFlags(gocbcore.BinaryType, gocbcore.NoCompression)
	case string:
		bytes = []byte(typeValue)
		flags = gocbcore.EncodeCommonFlags(gocbcore.StringType, gocbcore.NoCompression)
	case *string:
		bytes = []byte(*typeValue)
		flags = gocbcore.EncodeCommonFlags(gocbcore.StringType, gocbcore.NoCompression)
	case *interface{}:
		return t.Encode(*typeValue)
	default:
		bytes, err = t.serializer.Serialize(value)
		if err != nil {
			return nil, 0, err
		}
		flags = gocbcore.EncodeCommonFlags(gocbcore.JsonType, gocbcore.NoCompression)
	}

	// No compression supported currently

	return bytes, flags, nil
}

// DefaultJSONSerializer implements the JSONSerializer interface using json.Marshal/Unmarshal.
type DefaultJSONSerializer struct {
}

// Serialize applies the json.Marshal behaviour to serialize a Go type
func (s *DefaultJSONSerializer) Serialize(value interface{}) ([]byte, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// Deserialize applies the json.Unmarshal behaviour to deserialize into a Go type
func (s *DefaultJSONSerializer) Deserialize(bytes []byte, out interface{}) error {
	err := json.Unmarshal(bytes, &out)
	if err != nil {
		return err
	}

	return nil
}
