package candlestick

import (
	"time"
)

// Symbol represents a trading pair
type Symbol string

const (
	BTCUSDT  Symbol = "BTCUSDT"
	ETHUSDT  Symbol = "ETHUSDT"
	PEPEUSDT Symbol = "PEPEUSDT"
)

// Tick represents a single price update from the exchange
type Tick struct {
	Symbol    Symbol    `json:"symbol" gorm:"column:symbol"`
	Price     float64   `json:"price" gorm:"column:price"`
	Quantity  float64   `json:"quantity" gorm:"column:quantity"`
	Timestamp time.Time `json:"timestamp" gorm:"column:timestamp"`
}

// OHLC represents a candlestick with open, high, low, and close prices
type OHLC struct {
	Symbol    Symbol    `json:"symbol" gorm:"column:symbol"`
	Open      float64   `json:"open" gorm:"column:open"`
	High      float64   `json:"high" gorm:"column:high"`
	Low       float64   `json:"low" gorm:"column:low"`
	Close     float64   `json:"close" gorm:"column:close"`
	Volume    float64   `json:"volume" gorm:"column:volume"`
	OpenTime  time.Time `json:"open_time" gorm:"column:open_time"`
	CloseTime time.Time `json:"close_time" gorm:"column:close_time"`
}

// Aggregator defines the interface for OHLC data aggregation
type Aggregator interface {
	// Process handles a new tick and returns a completed OHLC if available
	Process(tick Tick) (*OHLC, error)
	// Current returns the current in-progress OHLC
	Current() *OHLC
}

// Storage defines the interface for OHLC data persistence
type Storage interface {
	// Store persists an OHLC candlestick
	Store(ohlc *OHLC) error
	// GetRange retrieves OHLC candlesticks for a symbol within a time range
	GetRange(symbol Symbol, start, end time.Time) ([]*OHLC, error)
}

// Streamer defines the interface for real-time OHLC data streaming
type Streamer interface {
	// Stream broadcasts an OHLC update to connected clients
	Stream(ohlc *OHLC) error
	// Subscribe returns a channel for receiving OHLC updates
	Subscribe(symbol Symbol) (<-chan *OHLC, error)
	// Unsubscribe removes a subscription
	Unsubscribe(symbol Symbol) error
}
