package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/gorilla/websocket"
)

// Binance WebSocket endpoints for failover
var websocketEndpoints = []string{
	"wss://stream.binance.com:9443/ws",
	"wss://stream-alt1.binance.com:9443/ws",
	"wss://stream-alt2.binance.com:9443/ws",
}

// KlineMessage represents the kline/candlestick websocket message format from Binance
type KlineMessage struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	Kline     struct {
		StartTime    int64  `json:"t"`
		EndTime      int64  `json:"T"`
		Symbol       string `json:"s"`
		Interval     string `json:"i"`
		FirstTradeID int64  `json:"f"`
		LastTradeID  int64  `json:"L"`
		Open         string `json:"o"`
		Close        string `json:"c"`
		High         string `json:"h"`
		Low          string `json:"l"`
		Volume       string `json:"v"`
		NumTrades    int64  `json:"n"`
		IsClosed     bool   `json:"x"`
		QuoteVolume  string `json:"q"`
	} `json:"k"`
}

// Client handles communication with Binance API through WebSocket
type Client struct {
	conn      *websocket.Conn
	mu        sync.RWMutex
	handlers  map[candlestick.Symbol][]chan<- candlestick.Tick
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// NewClient creates a new Binance WebSocket client
func NewClient(ctx context.Context) *Client {
	cctx, cancel := context.WithCancel(ctx)
	return &Client{
		handlers:  make(map[candlestick.Symbol][]chan<- candlestick.Tick),
		ctx:       cctx,
		cancelCtx: cancel,
	}
}

// Connect establishes a websocket connection to Binance with retry mechanism and endpoint failover
func (c *Client) Connect(symbols []candlestick.Symbol) error {
	// Create subscription string for multiple symbols with kline stream
	params := make([]string, len(symbols))
	for i, symbol := range symbols {
		// Convert symbol to lowercase as Binance requires
		symbolStr := string(symbol)
		params[i] = fmt.Sprintf("%s@kline_1m", symbolStr)
	}

	log.Printf("Subscribing to streams: %v", params)

	subRequest := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": params,
		"id":     1,
	}

	// Configure websocket dialer with timeouts
	dialer := websocket.Dialer{
		HandshakeTimeout:  10 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
	}

	// Implement retry with exponential backoff
	maxRetries := 5
	baseDelay := time.Second

	for endpointIndex, wsEndpoint := range websocketEndpoints {
		for retry := 0; retry < maxRetries; retry++ {
			// Check for context cancellation
			select {
			case <-c.ctx.Done():
				return fmt.Errorf("connection cancelled")
			default:
			}

			// Calculate backoff delay
			delay := baseDelay * time.Duration(1<<uint(retry))
			if retry > 0 {
				log.Printf("Retrying connection to %s (attempt %d/%d) after %v delay...", wsEndpoint, retry+1, maxRetries, delay)
				// Use timer instead of Sleep to handle cancellation
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
				case <-c.ctx.Done():
					timer.Stop()
					return fmt.Errorf("connection cancelled during retry delay")
				}
			}

			// Connect to websocket with context timeout
			dialCtx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
			conn, _, err := dialer.DialContext(dialCtx, wsEndpoint, nil)
			cancel()

			if err != nil {
				log.Printf("Connection attempt %d to %s failed: %v", retry+1, wsEndpoint, err)
				if retry == maxRetries-1 && endpointIndex == len(websocketEndpoints)-1 {
					return fmt.Errorf("websocket dial error after trying all endpoints: %v", err)
				}
				continue
			}

			// Set connection
			c.conn = conn

			// Configure connection parameters
			c.conn.SetReadLimit(65536)
			c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			c.conn.SetPongHandler(func(string) error {
				c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				return nil
			})

			// Subscribe to streams
			if err := conn.WriteJSON(subRequest); err != nil {
				conn.Close()
				if retry == maxRetries-1 && endpointIndex == len(websocketEndpoints)-1 {
					return fmt.Errorf("subscription request error after trying all endpoints: %v", err)
				}
				continue
			}

			// Start message handling and heartbeat
			go c.handleMessages()
			go c.maintainConnection()

			log.Printf("Successfully connected to %s", wsEndpoint)
			return nil
		}
	}

	return fmt.Errorf("failed to establish connection after trying all endpoints")
}

// Subscribe adds a handler for a specific symbol
func (c *Client) Subscribe(symbol candlestick.Symbol, ch chan<- candlestick.Tick) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[symbol] = append(c.handlers[symbol], ch)
}

// Close closes the websocket connection and stops all handlers
func (c *Client) Close() error {
	c.cancelCtx()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// maintainConnection sends periodic pings and handles reconnection
func (c *Client) maintainConnection() {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-pingTicker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Printf("ping error: %v", err)
				// Trigger reconnection
				go c.reconnect()
				return
			}
		}
	}
}

// reconnect attempts to reestablish the connection
func (c *Client) reconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
	}

	// Get current symbols
	var symbols []candlestick.Symbol
	for symbol := range c.handlers {
		symbols = append(symbols, symbol)
	}

	// Attempt to reconnect
	if err := c.Connect(symbols); err != nil {
		log.Printf("reconnection failed: %v", err)
	}
}

// handleMessages processes incoming websocket messages
func (c *Client) handleMessages() {
	for {
		select {
		case <-c.ctx.Done():
			log.Printf("INFO: Stopping message handler due to context cancellation")
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("ERROR: WebSocket read error: %v", err)
				go c.reconnect()
				return
			}

			log.Printf("DEBUG: Raw message received: %s", string(message))

			// Check for subscription response
			var subResp map[string]interface{}
			if err = json.Unmarshal(message, &subResp); err == nil {
				if _, ok := subResp["result"]; ok {
					log.Printf("INFO: Subscription confirmed with ID: %v", subResp["id"])
					continue
				}
			}

			var klineMsg KlineMessage
			if err = json.Unmarshal(message, &klineMsg); err != nil {
				log.Printf("ERROR: Message unmarshal error: %v, raw message: %s", err, string(message))
				continue
			}

			// Skip non-kline messages
			if klineMsg.EventType != "kline" {
				log.Printf("DEBUG: Skipping non-kline message type: %s", klineMsg.EventType)
				continue
			}

			// Convert message to Tick with error handling
			symbol := candlestick.Symbol(klineMsg.Symbol)
			price, err := strconv.ParseFloat(klineMsg.Kline.Close, 64)
			if err != nil {
				log.Printf("ERROR: Failed parsing price for %s: %v", symbol, err)
				continue
			}

			volume, err := strconv.ParseFloat(klineMsg.Kline.Volume, 64)
			if err != nil {
				log.Printf("ERROR: Failed parsing volume for %s: %v", symbol, err)
				continue
			}

			tick := candlestick.Tick{
				Symbol:    symbol,
				Price:     price,
				Quantity:  volume,
				Timestamp: time.Unix(0, klineMsg.Kline.EndTime*int64(time.Millisecond)),
			}

			log.Printf("INFO: Created tick for %s: price=%.2f volume=%.2f timestamp=%s",
				symbol, price, volume, tick.Timestamp.Format(time.RFC3339))

			// Distribute tick to handlers
			c.mu.RLock()
			handlers := c.handlers[symbol]
			handlerCount := len(handlers)
			c.mu.RUnlock()

			if handlerCount == 0 {
				log.Printf("WARNING: No handlers registered for symbol %s", symbol)
				continue
			}

			log.Printf("DEBUG: Attempting to distribute tick to %d handlers for %s", handlerCount, symbol)

			for i, handler := range handlers {
				select {
				case handler <- tick:
					log.Printf("DEBUG: Successfully sent tick to handler %d/%d for %s", i+1, handlerCount, symbol)
				default:
					log.Printf("WARNING: Handler %d/%d channel full for %s, dropping tick", i+1, handlerCount, symbol)
				}
			}
		}
	}
}
