package def

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"
)

type NoneSerialize struct {
}

func (s *NoneSerialize) Manual(v any) ([]byte, error) {
	return nil, nil
}
func (s *NoneSerialize) UnManual(data []byte, v any) error {
	return nil
}

type JsonSerialize struct {
}

func (s *JsonSerialize) Manual(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JsonSerialize) UnManual(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type ProtoSerialize struct {
}

func (s *ProtoSerialize) Manual(v any) ([]byte, error) {
	msg, err := v.(proto.Message)
	if !err {
		return nil, errors.New("protoSerialize: v is not proto.Message")
	}
	return proto.Marshal(msg)
}

func (s *ProtoSerialize) UnManual(data []byte, v any) error {
	msg, err := v.(proto.Message)
	if !err {
		return errors.New("protoSerialize: v is not proto.Message")
	}
	return proto.Unmarshal(data, msg)
}

type BytesSerialize struct {
}

func (s *BytesSerialize) Manual(v any) ([]byte, error) {
	return v.([]byte), nil
}
