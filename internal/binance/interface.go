package binance

import "github.com/azanium/ohlc/internal/candlestick"

// BinanceClient defines the interface for interacting with Binance
type BinanceClient interface {
	Connect(symbols []candlestick.Symbol) error
	Subscribe(symbol candlestick.Symbol, ch chan<- candlestick.Tick)
	Close() error
}
