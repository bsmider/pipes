package orchestrator

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bsmider/vibe/core/factory"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

type Orchestrator struct {
	pools            map[string]*WorkerPool
	poolsMu          sync.RWMutex
	responseChannels sync.Map // Map[packetID]chan *factory.IOPacket
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		pools: make(map[string]*WorkerPool),
		// responseChannels: make(map[string]chan *factory.IOPacket), ... instantiates itself
	}
}

// --- Lifecycle Management ---

func (o *Orchestrator) Spawn(processType string, binaryPath string, count int) error {
	for i := 0; i < count; i++ {
		// 1. Create Socketpair
		fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
		if err != nil {
			return err
		}

		uuid := uuid.New().String()
		id := fmt.Sprintf("%s-%.4s", processType, uuid)
		cmd := exec.Command(binaryPath, "--id", id)
		workerSide := os.NewFile(uintptr(fds[1]), "worker-socket")
		cmd.ExtraFiles = []*os.File{workerSide}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			workerSide.Close()
			return err
		}
		// Close parent's copy of the child's end
		workerSide.Close()

		// 3. Prepare Parent Connection
		orchSide := os.NewFile(uintptr(fds[0]), "orch-socket")
		conn, err := net.FileConn(orchSide)
		if err != nil {
			return err
		}

		mailbox := make(chan *factory.Packet)
		worker := NewWorker(id, processType, binaryPath, conn, cmd, mailbox)

		// 4. Register in Pool
		o.poolsMu.Lock()
		if _, ok := o.pools[processType]; !ok {
			o.pools[processType] = NewWorkerPool([]*Worker{}, 3*time.Second, 1)
		}
		o.pools[processType].workers = append(o.pools[processType].workers, worker)
		o.poolsMu.Unlock()

		// 5. Start the Listen Loop for this specific worker
		go worker.listen()
		go o.handleWorkerMailbox(worker)
	}
	return nil
}

func (o *Orchestrator) handleWorkerMailbox(worker *Worker) {
	for packet := range worker.mailbox {
		packet.Context.AddHop("orchestrator")
		if packet.Type == factory.PacketType_PACKET_TYPE_REQUEST {
			go o.handleInternalRequest(worker, packet)
		}

		if packet.Type == factory.PacketType_PACKET_TYPE_RESPONSE {
			o.routeResponse(packet)
		}
	}
}

func (o *Orchestrator) RouteRequest(packet *factory.Packet) (*factory.Packet, error) {
	o.poolsMu.RLock()
	pool, exists := o.pools[packet.TargetIoType]
	o.poolsMu.RUnlock()

	// The .4 before the 's' limits the output to 4 characters

	log.Printf("[Orchestrator] Routing request for %.4s (%s) to %s", packet.Id, packet.Type, packet.TargetIoType)

	if !exists {
		return nil, fmt.Errorf("no workers available for target type: %s", packet.TargetIoType)
	}

	var lastErr error

	// The retry loop: attempt 0 is the first try, then up to pool.retries
	for attempt := 0; attempt <= pool.retries; attempt++ {

		// 1. Select a worker for this specific attempt
		worker := pool.SelectWorker()
		if worker == nil {
			lastErr = fmt.Errorf("pool %s has no active workers", packet.TargetIoType)
			continue
		}

		// 2. Setup the response tracking channel
		// We use a buffer of 1 so the 'RouteResponse' logic doesn't block
		// if this loop has already timed out.
		respChan := make(chan *factory.Packet, 1)
		o.responseChannels.Store(packet.Id, respChan)

		// 3. Dispatch the packet
		if err := worker.sendPacket(packet); err != nil {
			o.responseChannels.Delete(packet.Id)
			lastErr = fmt.Errorf("worker %s send error: %w", worker.id, err)
			continue // Try next attempt with a different worker
		}

		// 4. Wait for response OR timeout
		//
		select {
		case response := <-respChan:
			// SUCCESS: Cleanup and return the result
			o.responseChannels.Delete(packet.Id)
			return response, nil

		case <-time.After(pool.timeout):
			// TIMEOUT: Cleanup and log
			o.responseChannels.Delete(packet.Id)
			lastErr = fmt.Errorf("attempt %d: timed out after %v", attempt, pool.timeout)
			// Loop continues to next retry
		}
	}

	return nil, fmt.Errorf("request failed after %d retries. Last error: %v", pool.retries, lastErr)
}

func (o *Orchestrator) handleInternalRequest(requester *Worker, packet *factory.Packet) (*factory.Packet, error) {
	o.poolsMu.RLock()
	pool, exists := o.pools[packet.TargetIoType]
	o.poolsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no workers available for target type: %s", packet.TargetIoType)
	}

	var lastErr error

	// The retry loop: attempt 0 is the first try, then up to pool.retries
	for attempt := 0; attempt <= pool.retries; attempt++ {

		// 1. Select a worker for this specific attempt
		worker := pool.SelectWorker()
		if worker == nil {
			lastErr = fmt.Errorf("pool %s has no active workers", packet.TargetIoType)
			continue
		}

		// 2. Setup the response tracking channel
		// We use a buffer of 1 so the 'RouteResponse' logic doesn't block
		// if this loop has already timed out.
		respChan := make(chan *factory.Packet, 1)
		o.responseChannels.Store(packet.Id, respChan)

		// 3. Dispatch the packet
		if err := worker.sendPacket(packet); err != nil {
			o.responseChannels.Delete(packet.Id)
			lastErr = fmt.Errorf("worker %s send error: %w", worker.id, err)
			continue // Try next attempt with a different worker
		}

		// 4. Wait for response OR timeout
		select {
		case response := <-respChan:
			// SUCCESS: Cleanup and return the result
			requester.sendPacket(response)
			o.responseChannels.Delete(packet.Id)
			return nil, nil

		case <-time.After(pool.timeout):
			// TIMEOUT: Cleanup and log
			o.responseChannels.Delete(packet.Id)
			lastErr = fmt.Errorf("attempt %d: timed out after %v", attempt, pool.timeout)
			// Loop continues to next retry
		}
	}

	return nil, fmt.Errorf("request failed after %d retries. Last error: %v", pool.retries, lastErr)
}

func (o *Orchestrator) routeResponse(packet *factory.Packet) error {
	responseChannel, ok := o.responseChannels.Load(packet.Id)
	if !ok {
		return fmt.Errorf("no response channel found for packet ID: %s", packet.Id)
	}

	ch, ok := responseChannel.(chan *factory.Packet)
	if !ok {
		return fmt.Errorf("response channel for packet ID %s is not a channel", packet.Id)
	}

	select {
	case ch <- packet:
		return nil
	default:
		return fmt.Errorf("requester channel for ID %s is full", packet.Id)
	}
}
