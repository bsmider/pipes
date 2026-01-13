package orchestrator

import (
	"bufio"
	"io"
	"log"
	"net"
	"os/exec"
	"sync"

	"github.com/bsmider/vibe/core/factory"
	"github.com/bsmider/vibe/core/factory/utils"
)

type Worker struct {
	id          string
	processType string
	binaryPath  string
	conn        net.Conn
	cmd         *exec.Cmd // So we can Kill() it if it freezes
	mailbox     chan *factory.Packet
	writeMu     *sync.Mutex
}

func NewWorker(id string, processType string, binaryPath string, conn net.Conn, cmd *exec.Cmd, mailbox chan *factory.Packet) *Worker {
	return &Worker{
		id:          id,
		processType: processType,
		binaryPath:  binaryPath,
		conn:        conn,
		cmd:         cmd,
		mailbox:     mailbox,
		writeMu:     &sync.Mutex{},
	}
}

func (w *Worker) listen() {
	defer func() {
		w.conn.Close()   // close the connection to the binary/process
		close(w.mailbox) // close the mailbox channel to the orchestrator
		log.Printf("[Orchestrator] Worker %s (PID %d) connection closed", w.id, w.cmd.Process.Pid)
	}()

	reader := bufio.NewReader(w.conn)
	for {
		packet := &factory.Packet{}
		if err := utils.ReadMessage(reader, packet); err != nil {
			if err != io.EOF {
				log.Printf("Error reading from %s: %v", w.id, err)
			}
			break
		}

		w.mailbox <- packet
	}
}

func (w *Worker) sendPacket(packet *factory.Packet) error {
	return utils.WriteMessage(w.conn, w.writeMu, packet)
}
