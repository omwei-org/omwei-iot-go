package omwei

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// localModbusTCPServer is a deterministic localhost simulator. It speaks the
// Modbus TCP MBAP envelope and serves a small synthetic SunSpec model chain.
type localModbusTCPServer struct {
	listener net.Listener
	registers map[uint16]uint16
	unitID uint8
	mu sync.Mutex
	serveErr error
	wg sync.WaitGroup
}

func startLocalModbusTCPServer(t *testing.T) *localModbusTCPServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &localModbusTCPServer{
		listener: listener,
		unitID: 7,
		registers: map[uint16]uint16{
			100: 103, // model ID
			101: 14,  // model data length
			114: 1234, // W raw value
			115: 0xffff, // W_SF = -1
		},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer conn.Close()
				if err := s.serveOne(conn); err != nil {
					s.mu.Lock()
					if s.serveErr == nil {
						s.serveErr = err
					}
					s.mu.Unlock()
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		s.wg.Wait()
		s.mu.Lock()
		err := s.serveErr
		s.mu.Unlock()
		if err != nil {
			t.Errorf("local Modbus TCP simulator: %v", err)
		}
	})
	return s
}

func (s *localModbusTCPServer) address() string { return s.listener.Addr().String() }

func (s *localModbusTCPServer) serveOne(conn net.Conn) error {
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	transactionID := binary.BigEndian.Uint16(header[0:2])
	protocolID := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	unitID := header[6]
	if protocolID != 0 || length != 6 || unitID != s.unitID {
		return fmt.Errorf("invalid request MBAP protocol=%d length=%d unit=%d", protocolID, length, unitID)
	}
	pdu := make([]byte, 5)
	if _, err := io.ReadFull(conn, pdu); err != nil {
		return err
	}
	function := pdu[0]
	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	if (function != 3 && function != 4) || quantity == 0 || quantity > 125 {
		return fmt.Errorf("invalid read request function=%d quantity=%d", function, quantity)
	}
	response := make([]byte, 9+int(quantity)*2)
	binary.BigEndian.PutUint16(response[0:2], transactionID)
	binary.BigEndian.PutUint16(response[2:4], 0)
	binary.BigEndian.PutUint16(response[4:6], uint16(3+quantity*2))
	response[6] = unitID
	response[7] = function
	response[8] = byte(quantity * 2)
	s.mu.Lock()
	for i := uint16(0); i < quantity; i++ {
		value := s.registers[start+i]
		binary.BigEndian.PutUint16(response[9+int(i)*2:11+int(i)*2], value)
	}
	s.mu.Unlock()
	_, err := conn.Write(response)
	return err
}

// localModbusTCPClient is a minimal test-only Modbus TCP client. Production
// code should use a maintained Modbus client; this helper exists to verify the
// adapter against actual TCP framing without adding a runtime dependency.
type localModbusTCPClient struct {
	address string
	nextTransactionID uint16
	mu sync.Mutex
}

func (c *localModbusTCPClient) ReadHoldingRegisters(ctx context.Context, unitID uint8, address uint16, quantity uint16) ([]uint16, error) {
	return c.read(ctx, unitID, address, quantity, 3)
}

func (c *localModbusTCPClient) ReadInputRegisters(ctx context.Context, unitID uint8, address uint16, quantity uint16) ([]uint16, error) {
	return c.read(ctx, unitID, address, quantity, 4)
}

func (c *localModbusTCPClient) read(ctx context.Context, unitID uint8, address uint16, quantity uint16, function byte) ([]uint16, error) {
	c.mu.Lock()
	c.nextTransactionID++
	transactionID := c.nextTransactionID
	c.mu.Unlock()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], transactionID)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = unitID
	request[7] = function
	binary.BigEndian.PutUint16(request[8:10], address)
	binary.BigEndian.PutUint16(request[10:12], quantity)
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}
	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(header[0:2]) != transactionID ||
		binary.BigEndian.Uint16(header[2:4]) != 0 ||
		header[6] != unitID {
		return nil, fmt.Errorf("invalid Modbus TCP response header")
	}
	length := binary.BigEndian.Uint16(header[4:6])
	if length < 3 || length > 253 {
		return nil, fmt.Errorf("invalid Modbus TCP response length %d", length)
	}
	pdu := make([]byte, int(length)-1)
	if _, err := io.ReadFull(conn, pdu); err != nil {
		return nil, err
	}
	if pdu[0] == function|0x80 {
		return nil, fmt.Errorf("Modbus exception response code %d", pdu[1])
	}
	if pdu[0] != function || len(pdu) < 2 || int(pdu[1]) != int(quantity)*2 || len(pdu) != 2+int(quantity)*2 {
		return nil, fmt.Errorf("invalid Modbus read response function=%d bytes=%d payload=%d", pdu[0], pdu[1], len(pdu))
	}
	values := make([]uint16, quantity)
	for i := range values {
		values[i] = binary.BigEndian.Uint16(pdu[2+i*2 : 4+i*2])
	}
	return values, nil
}

func TestSunSpecModel103OverLocalModbusTCPToEABCAtom(t *testing.T) {
	server := startLocalModbusTCPServer(t)
	client := &localModbusTCPClient{address: server.address()}
	source := SourceIdentity{Protocol: "sunspec-modbus", Device: "simulated-inverter-7", Point: "model:103:W;scale-factor:W_SF"}
	adapter, err := DiscoverSunSpecModel103Adapter(context.Background(), client, source, 7, SunSpecInputRegister,
		SunSpecModelDiscoveryConfig{FirstHeaderAddress: 100, MaxModels: 16})
	if err != nil {
		t.Fatal(err)
	}
	mapping := SourceMapping{
		Source: source, DescriptorID: 69,
		SourceUnit: "watt (after W_SF application)", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
	result, err := ReadAndMapSunSpecModel103ActivePower(adapter, mapping)
	if err != nil {
		t.Fatal(err)
	}
	if result.Atom.CanonicalValue != 123 {
		t.Fatalf("want 123 W from simulated raw W=1234 and W_SF=-1, got %d", result.Atom.CanonicalValue)
	}
}
