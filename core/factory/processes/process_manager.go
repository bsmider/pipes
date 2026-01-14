package processes

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"log"

	"github.com/bsmider/pipes/core/factory"
	"github.com/bsmider/pipes/core/factory/utils"
	"google.golang.org/protobuf/proto"
)

// used to help manage a singular process
// it handles inputs, outputs, parallelism, and multiplexing
type IONode struct {
	id               string
	mapMu            sync.Mutex                      // used to synchronize access to the map
	writeMu          sync.Mutex                      // used to synchronize writes to the writer
	ResponseChannels map[string]chan *factory.Packet // maps an id to a channel that made an outbound call and is awaiting a response
	RequestChannel   chan *factory.Packet            // a channel that processes new requests
	reader           io.Reader                       // the reader to use for reading input from io
	writer           io.Writer                       // the writer to use for writing output to io
	conn             net.Conn                        // the connection to use for reading and writing
}

var (
	instance *IONode
	once     sync.Once
	started  sync.Once
)

// GetInstance provides the singleton instance.
// It uses sync.Once to handle thread-safe initialization.
func GetIONode(id ...string) *IONode {
	once.Do(func() {
		var reader io.Reader = os.Stdin
		var writer io.Writer = os.Stdout
		var socketConn net.Conn

		// --- SOCKETPAIR DETECTION LOGIC ---
		// By convention, ExtraFiles[0] passed by the parent becomes FD 3.
		// We attempt to create a net.Conn from File Descriptor 3.
		f := os.NewFile(3, "vibe-socket")
		if f != nil {
			// Check if FD 3 is actually a valid socket
			conn, err := net.FileConn(f)
			if err == nil {
				// log.Println("[IONode] High-Performance Socketpair detected on FD 3")
				socketConn = conn
				// In a socketpair, the same handle is used for BOTH read and write
				reader = conn
				writer = conn
			} else {
				log.Println("[IONode] No socketpair detected, falling back to Stdio")
				f.Close()
			}
		}

		var finalID string
		if len(id) > 0 {
			finalID = id[0]
		} else {
			finalID = "unknown-node"
		}

		instance = &IONode{
			id:               finalID,
			mapMu:            sync.Mutex{},
			writeMu:          sync.Mutex{},
			ResponseChannels: make(map[string]chan *factory.Packet), // maps id's to channels that sent a request to another binary and are waiting for a response
			RequestChannel:   make(chan *factory.Packet, 100),       // processes new requests
			reader:           reader,
			writer:           writer,
			conn:             socketConn,
		}
	})
	return instance
}

// AwaitPackets starts a background process to read packets from Stdin.
// It should only be called once.
func (node *IONode) readInput() {
	reader := bufio.NewReader(node.reader)

	for {
		packet := &factory.Packet{}

		// Read a length-prefixed PipeMessage
		if err := utils.ReadMessage(reader, packet); err != nil {
			// 1. The stream ended intentionally. Exit quietly.
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}

			// 2. Something actually went wrong. Log and bail.
			log.Printf("[ProcessRunner] FATAL: Stream corrupted: %v\n", err)
			return
		}

		packet.Context.AddHop(node.id)

		node.routePacket(packet)
	}
}

// Listen starts the background process to read packets from Stdin.
// It should only be called once.
func (node *IONode) Listen() {
	started.Do(func() {
		go node.readInput()
	})
}

func (node *IONode) routePacket(packet *factory.Packet) {
	// MULTIPLEXING LOGIC
	node.mapMu.Lock()
	responseChannel, isAwaitingResponse := node.ResponseChannels[packet.Id]
	node.mapMu.Unlock()

	if isAwaitingResponse {
		// CASE A: RESPONSE to a request we sent
		select {
		case responseChannel <- packet:
			// Success
		default:
			log.Printf("[ProcessRunner] Warning: Drop packet %s - channel full/no receiver\n", packet.Id)
		}
	} else {
		// CASE B: NEW REQUEST from another process
		select {
		case node.RequestChannel <- packet:
			// log.Printf("[ProcessRunner] Routing ID %s to NewRequestChannel\n", packet.Id)
		default:
			log.Printf("[ProcessRunner] Critical: RequestChannel full, dropping packet %s\n", packet.Id)
		}
	}
}

func Handle[RequestPayloadType proto.Message, ResponsePayloadType proto.Message](
	logic func(context.Context, RequestPayloadType) (ResponsePayloadType, error),
) {
	node := GetIONode()

	go func() {
		for requestPacket := range node.RequestChannel {
			go func(requestPacket *factory.Packet) {
				requestObject, err := utils.BytesToType[RequestPayloadType](requestPacket.Payload)
				if err != nil {
					log.Printf("decode error: %v", err)
					return
				}

				context, cancel := requestPacket.Context.ToGoContext()
				defer cancel()

				responseObject, err := logic(context, requestObject)

				select {
				case <-context.Done():
					log.Println("Client canceled or timeout reached, skipping response")
					return
				default:
					// Continue to send response
				}

				respErr := (&factory.Error{}).FromGoError(err)
				respContext := (&factory.Context{}).FromGoContext(context)
				responsePacket, err := factory.CreateResponsePacket(requestPacket.Id, "", respContext, responseObject, respErr)
				if err != nil {
					log.Printf("encode error: %v", err)
					return
				}

				err = node.sendPacket(responsePacket)
				if err != nil {
					log.Printf("write error: %v", err)
				}
			}(requestPacket)
		}
	}()
}

// sends a packet to stdout
func (node *IONode) sendPacket(packet *factory.Packet) error {
	return utils.WriteMessage(node.writer, &node.writeMu, packet)
}

// Sends a request to another process and blocks until a response is received
func (node *IONode) executeRequest(packet *factory.Packet) (*factory.Packet, error) {
	responseChannel := make(chan *factory.Packet, 1)

	// 1. Register our ID
	node.mapMu.Lock()
	node.ResponseChannels[packet.Id] = responseChannel
	node.mapMu.Unlock()

	// 2. Ensure cleanup
	defer func() {
		node.mapMu.Lock()
		delete(node.ResponseChannels, packet.Id)
		node.mapMu.Unlock()
	}()

	// 3. Send via the runner's internal WriteRequest
	err := node.sendPacket(packet)
	if err != nil {
		return nil, err
	}

	// 4. Block for response
	return node.awaitResponse(responseChannel, 10*time.Second)
}

func (node *IONode) awaitResponse(responseChannel chan *factory.Packet, timeout time.Duration) (*factory.Packet, error) {
	select {
	case response := <-responseChannel:
		return response, nil
	case <-time.After(timeout):
		return nil, errors.New("request timed out")
	}
}

func Call[RequestType proto.Message, ResponseType proto.Message](targetIoType string, context context.Context, payload RequestType) (ResponseType, error) {
	var zero ResponseType

	ioCtx := (&factory.Context{}).FromGoContext(context)
	requestPacket, err := factory.CreateRequestPacket(targetIoType, ioCtx, payload, nil)
	if err != nil {
		return zero, err
	}

	responsePacket, err := GetIONode().executeRequest(requestPacket)
	if err != nil {
		return zero, err
	}

	// converts the payload bytes to a ResponseType
	out, err := utils.BytesToType[ResponseType](responsePacket.Payload)
	if err != nil {
		return zero, err
	}

	// Merge hops from the response back into our current context
	factory.UpdateContext(context, responsePacket.Context)

	packetError := responsePacket.Error.ToGoError()
	return out, packetError
}
