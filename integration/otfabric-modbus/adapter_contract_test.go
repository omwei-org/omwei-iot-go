package modbusadapter

import (
	"context"
	"testing"

	omwei "github.com/omwei-org/omwei-iot-go"
	"github.com/otfabric/modbus"
)

// Compile-time contract: the pinned upstream client's public methods must
// remain compatible with the transport-neutral EABC adapter interface.
var _ omwei.SunSpecModel103RegisterClient = (*modbus.Client)(nil)

func TestProductionClientFitsAdapterContract(t *testing.T) {
	var client *modbus.Client
	var registerClient omwei.SunSpecModel103RegisterClient = client
	if registerClient != nil {
		t.Fatal("nil production client unexpectedly became non-nil")
	}
	// Compile-check the method signatures without opening a socket.
	var readInput func(context.Context, uint8, uint16, uint16) ([]uint16, error) = client.ReadInputRegisters
	var readHolding func(context.Context, uint8, uint16, uint16) ([]uint16, error) = client.ReadHoldingRegisters
	_, _ = readInput, readHolding
}
