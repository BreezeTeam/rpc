package json

import (
	"encoding/json"
	"rpc/codec"
)

type Marshaler struct{}

func (m Marshaler) Marshal(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (m Marshaler) Unmarshal(bytes []byte, i interface{}) error {
	return json.Unmarshal(bytes, i)
}

func (m Marshaler) String() string {
	return "json"
}

var _ codec.Marshaler = (*Marshaler)(nil)
