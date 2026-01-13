package utils

import (
	"reflect"

	"google.golang.org/protobuf/proto"
)

func SerializeMessage[T proto.Message](msg T) ([]byte, error) {
	return proto.Marshal(msg)
}

// converts bytes to a type
func BytesToType[Type proto.Message](payload []byte) (Type, error) {
	// Create a new instance of the underlying type that T represents
	msg := reflect.New(reflect.TypeOf(*new(Type)).Elem()).Interface().(Type)

	if err := proto.Unmarshal(payload, msg); err != nil {
		var zero Type
		return zero, err
	}
	return msg, nil
}
