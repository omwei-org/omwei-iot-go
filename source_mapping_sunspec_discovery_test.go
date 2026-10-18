package omwei

import (
	"context"
	"errors"
	"testing"
)

type fakeSunSpecDiscoveryClient struct {
	responses [][]uint16
	addrs []uint16
	quantities []uint16
	err error
}

func (f *fakeSunSpecDiscoveryClient) ReadHoldingRegisters(_ context.Context, _ uint8, addr uint16, quantity uint16) ([]uint16, error) {
	f.addrs = append(f.addrs, addr)
	f.quantities = append(f.quantities, quantity)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected register read")
	}
	result := f.responses[0]
	f.responses = f.responses[1:]
	return result, nil
}

func (f *fakeSunSpecDiscoveryClient) ReadInputRegisters(ctx context.Context, unit uint8, addr uint16, quantity uint16) ([]uint16, error) {
	return f.ReadHoldingRegisters(ctx, unit, addr, quantity)
}

func TestDiscoverSunSpecModel103WalksHeaders(t *testing.T) {
	client := &fakeSunSpecDiscoveryClient{responses: [][]uint16{
		{1, 2},       // model 1 at 1000; data 1002..1003
		{103, 14},    // Model 103 header at 1004; data begins 1006
	}}
	header, err := DiscoverSunSpecModel103(context.Background(), client, 7, SunSpecInputRegister,
		SunSpecModelDiscoveryConfig{FirstHeaderAddress: 1000, MaxModels: 10})
	if err != nil {
		t.Fatal(err)
	}
	if header.ModelID != 103 || header.ModelLength != 14 || header.HeaderAddress != 1004 || header.StartAddress != 1006 {
		t.Fatalf("unexpected discovery result: %+v", header)
	}
	if len(client.addrs) != 2 || client.addrs[0] != 1000 || client.addrs[1] != 1004 ||
		client.quantities[0] != 2 || client.quantities[1] != 2 {
		t.Fatalf("unexpected header reads: addresses=%v quantities=%v", client.addrs, client.quantities)
	}
}

func TestDiscoverSunSpecModel103RejectsMalformedOrMissingModel(t *testing.T) {
	tests := []struct {
		name string
		responses [][]uint16
		max uint16
		want error
	}{
		{name: "short header", responses: [][]uint16{{103}}, max: 1, want: ErrInvalidSunSpecValue},
		{name: "zero length", responses: [][]uint16{{1, 0}}, max: 1, want: ErrInvalidSunSpecValue},
		{name: "short model 103", responses: [][]uint16{{103, 13}}, max: 1, want: ErrInvalidSunSpecValue},
		{name: "end marker", responses: [][]uint16{{0xffff, 0}}, max: 1, want: ErrInvalidSourceMapping},
		{name: "model limit", responses: [][]uint16{{1, 1}}, max: 1, want: ErrInvalidSourceMapping},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeSunSpecDiscoveryClient{responses: tt.responses}
			_, err := DiscoverSunSpecModel103(context.Background(), client, 1, SunSpecInputRegister,
				SunSpecModelDiscoveryConfig{FirstHeaderAddress: 100, MaxModels: tt.max})
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}

func TestDiscoverSunSpecModel103PropagatesTransportError(t *testing.T) {
	transportErr := errors.New("transport down")
	client := &fakeSunSpecDiscoveryClient{err: transportErr}
	_, err := DiscoverSunSpecModel103(context.Background(), client, 1, SunSpecInputRegister,
		SunSpecModelDiscoveryConfig{FirstHeaderAddress: 100, MaxModels: 2})
	if !errors.Is(err, transportErr) {
		t.Fatalf("got %v, want transport error", err)
	}
}
