package factory

import (
	"fmt"

	"github.com/bsmider/pipes/core/factory/utils"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func NewPacket(id string, packetType PacketType, targetIoType string, context *Context, payload []byte, error *Error) *Packet {
	return &Packet{
		Id:           id,
		Type:         packetType,
		TargetIoType: targetIoType,
		Context:      context,
		Payload:      payload,
		Error:        error,
	}
}

// serializes the message to bytes and creates a new Packet
func CreatePacket[PayloadType proto.Message](packetId string, packetType PacketType, targetIoType string, context *Context, payload PayloadType, err *Error) (*Packet, error) {
	// serializes the message to bytes
	bytes, serr := utils.SerializeMessage(payload)
	if serr != nil {
		return nil, fmt.Errorf("serialize error: %w", serr)
	}

	return NewPacket(packetId, packetType, targetIoType, context, bytes, err), nil
}

func CreateRequestPacket[PayloadType proto.Message](targetIoType string, context *Context, payload PayloadType, err *Error) (*Packet, error) {
	// generates an id for the packet
	packetId := GeneratePacketId()

	// creates a new Packet of packet type REQUEST
	return CreatePacket(packetId, PacketType_PACKET_TYPE_REQUEST, targetIoType, context, payload, err)
}

func CreateResponsePacket[PayloadType proto.Message](packetId string, targetIoType string, context *Context, payload PayloadType, err *Error) (*Packet, error) {
	// creates a new Packet of packet type RESPONSE
	return CreatePacket(packetId, PacketType_PACKET_TYPE_RESPONSE, targetIoType, context, payload, err)
}

func GeneratePacketId() string {
	return uuid.NewString()
}

// we might want to change this to better handle the BytesToType error
func DeserializePacket[PayloadType proto.Message](packet *Packet) (PayloadType, error) {
	err := packet.Error.ToGoError()
	payloadObj, bttErr := utils.BytesToType[PayloadType](packet.Payload)
	if bttErr != nil {
		var zero PayloadType
		return zero, bttErr
	}
	return payloadObj, err
}

// PrintDetails outputs the entire packet structure to the console in a readable format.
func (p *Packet) PrintDetails() {
	if p == nil {
		fmt.Println("[Packet] <nil>")
		return
	}

	// Configure the marshaler for "pretty printing"
	options := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: true, // Shows empty fields like error: null or payload: ""
	}

	jsonBytes, err := options.Marshal(p)
	if err != nil {
		fmt.Printf("[Packet Error] Could not format packet: %v\n", err)
		return
	}

	fmt.Printf("--- Packet Detail: %.4s ---\n%s\n---------------------------\n", p.Id, string(jsonBytes))
}
