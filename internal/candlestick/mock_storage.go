package candlestick

import (
	"sync"
	"time"
)

// mockStorage implements the Storage interface for testing
type mockStorage struct {
	mu    sync.RWMutex
	ticks []*Tick
	ohlcs []*OHLC
}

// NewMockStorage creates a new mock storage for testing
func NewMockStorage() Storage {
	return &mockStorage{
		ticks: make([]*Tick, 0),
		ohlcs: make([]*OHLC, 0),
	}
}

// Store implements Storage.Store
func (m *mockStorage) Store(ohlc *OHLC) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ohlcs = append(m.ohlcs, ohlc)
	return nil
}

// StoreTick implements Storage.StoreTick
func (m *mockStorage) StoreTick(tick *Tick) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ticks = append(m.ticks, tick)
	return nil
}

// GetRange implements Storage.GetRange
func (m *mockStorage) GetRange(symbol Symbol, start, end time.Time) ([]*OHLC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*OHLC
	for _, ohlc := range m.ohlcs {
		if ohlc.Symbol == symbol && ohlc.OpenTime.After(start) && ohlc.CloseTime.Before(end) {
			result = append(result, ohlc)
		}
	}
	return result, nil
}

// GetStoredTicks returns all stored ticks (helper for testing)
func (m *mockStorage) GetStoredTicks() []*Tick {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ticks
}
