package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/azanium/ohlc/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBinanceClient implements a mock version of the Binance client for testing
type MockBinanceClient struct {
	mu       sync.RWMutex
	handlers map[candlestick.Symbol][]chan<- candlestick.Tick
}

func NewMockBinanceClient() *MockBinanceClient {
	return &MockBinanceClient{
		handlers: make(map[candlestick.Symbol][]chan<- candlestick.Tick),
	}
}

func (m *MockBinanceClient) Connect(symbols []candlestick.Symbol) error {
	return nil
}

func (m *MockBinanceClient) Subscribe(symbol candlestick.Symbol, ch chan<- candlestick.Tick) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[symbol] = append(m.handlers[symbol], ch)
}

func (m *MockBinanceClient) Close() error {
	return nil
}

func (m *MockBinanceClient) SimulateTick(symbol candlestick.Symbol, price, quantity float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tick := candlestick.Tick{
		Symbol:    symbol,
		Price:     price,
		Quantity:  quantity,
		Timestamp: time.Now(),
	}

	for _, ch := range m.handlers[symbol] {
		ch <- tick
	}
}

func TestServiceIntegration(t *testing.T) {
	// Create mock client
	mockClient := NewMockBinanceClient()

	// Configure service
	config := service.Config{
		Symbols:        []candlestick.Symbol{"BTCUSDT"},
		Interval:       time.Second, // Use shorter interval for testing
		MaxSubscribers: 10,
		ChannelSize:    100,
		StorageDSN:     "postgres://demo:demo123@localhost:65432/ohlc?sslmode=disable",
	}

	// Create service with mock client
	svc, err := service.New(context.Background(), config)
	require.NoError(t, err)
	// Replace the real client with mock
	svc.SetClient(mockClient)
	defer svc.Stop()

	// Start service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = svc.Start(ctx)
	require.NoError(t, err)

	// Get streamer
	streamer := svc.GetStreamer()
	assert.NotNil(t, streamer)

	// Subscribe to OHLC updates
	updateCh := make(chan *candlestick.OHLC, 10)
	streamer.Subscribe("BTCUSDT", updateCh)
	require.NoError(t, err)

	// Wait for updates with a reasonable timeout
	select {
	case <-ctx.Done():
		t.Fatal("Test timed out waiting for OHLC update")
	case update := <-updateCh:
		// Verify OHLC data
		assert.NotNil(t, update)
		assert.Equal(t, "BTCUSDT", string(update.Symbol))
		assert.True(t, update.Open > 0, "Open price should be greater than 0")
		assert.True(t, update.High >= update.Open, "High should be >= Open")
		assert.True(t, update.Low <= update.Open, "Low should be <= Open")
		assert.True(t, update.Close > 0, "Close price should be greater than 0")
		assert.True(t, update.Volume > 0, "Volume should be greater than 0")
	}
}
