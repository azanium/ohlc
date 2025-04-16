package candlestick

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// aggregator implements the Aggregator interface for OHLC data
type aggregator struct {
	mu       sync.RWMutex
	current  map[Symbol]*OHLC
	interval time.Duration
	storage  Storage
}

// NewAggregator creates a new OHLC aggregator with the specified interval
func NewAggregator(interval time.Duration, storage Storage) Aggregator {
	return &aggregator{
		current:  make(map[Symbol]*OHLC),
		interval: interval,
		storage:  storage,
	}
}

// Process handles a new tick and returns a completed OHLC if available
func (a *aggregator) Process(tick Tick) (*OHLC, error) {
	log.Printf("Processing tick: symbol=%s, price=%.2f, quantity=%.2f, timestamp=%s",
		tick.Symbol, tick.Price, tick.Quantity, tick.Timestamp.Format(time.RFC3339))

	// Store the tick in the database
	if err := a.storage.StoreTick(&tick); err != nil {
		log.Printf("Error storing tick: %v", err)
		return nil, fmt.Errorf("failed to store tick: %v", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create current OHLC for the symbol
	ohlc, exists := a.current[tick.Symbol]
	if !exists || a.shouldStartNewCandle(tick.Timestamp, ohlc) {
		// If we have an existing OHLC, it's complete
		var completed *OHLC
		if exists {
			completed = ohlc
		}

		// Start a new candle
		startTime := tick.Timestamp.Truncate(a.interval)
		log.Printf("Starting new candle for symbol=%s at time=%s", tick.Symbol, startTime.Format(time.RFC3339))
		a.current[tick.Symbol] = &OHLC{
			Symbol:    tick.Symbol,
			Open:      tick.Price,
			High:      tick.Price,
			Low:       tick.Price,
			Close:     tick.Price,
			Volume:    tick.Quantity,
			OpenTime:  startTime,
			CloseTime: startTime.Add(a.interval),
		}

		return completed, nil
	}

	// Update current OHLC
	ohlc.High = max(ohlc.High, tick.Price)
	ohlc.Low = min(ohlc.Low, tick.Price)
	ohlc.Close = tick.Price
	ohlc.Volume += tick.Quantity

	return nil, nil
}

// Current returns the current in-progress OHLC
func (a *aggregator) Current() *OHLC {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent external modifications
	if len(a.current) == 0 {
		return nil
	}

	// Return the first available OHLC (since we typically track one symbol)
	for _, ohlc := range a.current {
		copy := *ohlc
		return &copy
	}

	return nil
}

// shouldStartNewCandle checks if it's time to start a new candlestick
func (a *aggregator) shouldStartNewCandle(timestamp time.Time, current *OHLC) bool {
	if current == nil {
		return true
	}
	return timestamp.After(current.CloseTime)
}

// Helper functions
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
