package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"
)

const HeaderSize = 4

// WriteMessage wraps a Protobuf message in a length-prefixed frame and writes it to w.
// The mu parameter ensures that the length and payload are written as one atomic block.
func WriteMessage(w io.Writer, mu *sync.Mutex, msg proto.Message) error {
	// log.Printf("[IO UTILS] Writing message")
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	lenBuf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(bytes)))

	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	// Write header then payload
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	_, err = w.Write(bytes)

	return err
}

// ReadMessage reads a length-prefixed Protobuf message from r.
func ReadMessage(r io.Reader, msg proto.Message) error {
	// log.Printf("[IO UTILS] READING MESSAGE")
	// 1. Read the header
	lenBuf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return err // Expected EOF if pipe closes
	}

	// 2. Parse length
	length := binary.BigEndian.Uint32(lenBuf)

	// 3. Read exactly 'length' bytes
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	// 4. Unmarshal into the provided pointer
	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	return nil
}
