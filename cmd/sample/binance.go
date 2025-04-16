package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

type BinanceCombinedStream struct {
	Stream string          `json:"stream"`
	Data   BinanceAggTrade `json:"data"`
}

type BinanceAggTrade struct {
	EventType    string `json:"e"` // aggTrade
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	TradeID      int64  `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstTradeID int64  `json:"f"`
	LastTradeID  int64  `json:"l"`
	TradeTime    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
	Ignore       bool   `json:"M"`
}

func main() {
	symbols := []string{"btcusdt", "ethusdt", "pepeusdt"}
	streams := ""
	for i, symbol := range symbols {
		streams += symbol + "@aggTrade"
		if i != len(symbols)-1 {
			streams += "/"
		}
	}

	url := fmt.Sprintf("wss://stream.binance.com:9443/stream?streams=%s", streams)
	log.Println("Connecting to:", url)

	header := http.Header{}
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatal("Error connecting to WebSocket:", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}

			var stream BinanceCombinedStream
			if err := json.Unmarshal(message, &stream); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}

			log.Printf("[%s] Price: %s, Quantity: %s\n",
				stream.Data.Symbol,
				stream.Data.Price,
				stream.Data.Quantity,
			)
		}
	}()

	// Wait for signal
	<-interrupt
	log.Println("Interrupt received, shutting down...")
	conn.Close()
}
