package candlestick

import (
	"testing"
	"time"
)

func TestNewAggregator(t *testing.T) {
	interval := time.Minute
	agg := NewAggregator(interval)
	if agg == nil {
		t.Fatal("Expected non-nil aggregator")
	}
}

func TestProcess(t *testing.T) {
	interval := time.Minute
	agg := NewAggregator(interval)

	tests := []struct {
		name      string
		tick      Tick
		expectNil bool
	}{
		{
			name: "First tick",
			tick: Tick{
				Symbol:    BTCUSDT,
				Price:     50000.0,
				Quantity:  1.0,
				Timestamp: time.Now(),
			},
			expectNil: true,
		},
		{
			name: "Higher price",
			tick: Tick{
				Symbol:    BTCUSDT,
				Price:     51000.0,
				Quantity:  0.5,
				Timestamp: time.Now(),
			},
			expectNil: true,
		},
		{
			name: "Lower price",
			tick: Tick{
				Symbol:    BTCUSDT,
				Price:     49000.0,
				Quantity:  1.5,
				Timestamp: time.Now().Add(interval),
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ohlc, err := agg.Process(tt.tick)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectNil && ohlc != nil {
				t.Error("Expected nil OHLC, got non-nil")
			}

			if !tt.expectNil {
				if ohlc == nil {
					t.Fatal("Expected non-nil OHLC")
				}
				if ohlc.Symbol != tt.tick.Symbol {
					t.Errorf("Expected symbol %s, got %s", tt.tick.Symbol, ohlc.Symbol)
				}
				if ohlc.Volume <= 0 {
					t.Error("Expected positive volume")
				}
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	interval := time.Minute
	agg := NewAggregator(interval)

	// Test empty state
	if current := agg.Current(); current != nil {
		t.Error("Expected nil current OHLC when no ticks processed")
	}

	// Process a tick
	tick := Tick{
		Symbol:    BTCUSDT,
		Price:     50000.0,
		Quantity:  1.0,
		Timestamp: time.Now(),
	}

	_, err := agg.Process(tick)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Get current candle
	current := agg.Current()
	if current == nil {
		t.Fatal("Expected non-nil current OHLC")
	}

	if current.Symbol != tick.Symbol {
		t.Errorf("Expected symbol %s, got %s", tick.Symbol, current.Symbol)
	}

	if current.Open != tick.Price {
		t.Errorf("Expected open price %f, got %f", tick.Price, current.Open)
	}

	if current.Volume != tick.Quantity {
		t.Errorf("Expected volume %f, got %f", tick.Quantity, current.Volume)
	}
}
