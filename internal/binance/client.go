package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/gorilla/websocket"
)

// Custom error types for better error handling
type ConnectionError struct {
	Endpoint string
	Attempt  int
	Err      error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("failed to connect to %s (attempt %d): %v", e.Endpoint, e.Attempt, e.Err)
}

type SubscriptionError struct {
	Symbols []candlestick.Symbol
	Err     error
}

func (e *SubscriptionError) Error() string {
	return fmt.Sprintf("failed to subscribe to symbols %v: %v", e.Symbols, e.Err)
}

type MessageParsingError struct {
	MessageType string
	RawMessage  string
	Err         error
}

func (e *MessageParsingError) Error() string {
	return fmt.Sprintf("failed to parse %s message: %v, raw: %s", e.MessageType, e.Err, e.RawMessage)
}

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

// AggTradeMessage represents the aggregated trade websocket message format from Binance
type AggTradeMessage struct {
	EventType    string `json:"e"`
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	ID           int64  `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstID      int64  `json:"f"`
	LastID       int64  `json:"l"`
	Timestamp    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
	IsBestMatch  bool   `json:"M"`
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
		symbolStr := strings.ToLower(string(symbol))
		params[i] = fmt.Sprintf("%s@aggTrade", symbolStr)
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
				connErr := &ConnectionError{Endpoint: wsEndpoint, Attempt: retry + 1, Err: err}
				log.Printf("Connection error: %v", connErr)
				if retry == maxRetries-1 && endpointIndex == len(websocketEndpoints)-1 {
					return fmt.Errorf("websocket dial error after exhausting all endpoints and retries: %v", err)
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
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic in handleMessages: %v", r)
						// Attempt to reconnect
						if err := c.Connect(symbols); err != nil {
							log.Printf("Failed to reconnect after panic: %v", err)
						}
					}
				}()
				c.handleMessages()
			}()
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic in maintainConnection: %v", r)
					}
				}()
				c.maintainConnection()
			}()

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
	// Signal all goroutines to stop
	c.cancelCtx()

	// Safely close the connection
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	// Close the connection outside the lock
	if conn != nil {
		log.Printf("INFO: Closing WebSocket connection")
		return conn.Close()
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
			log.Printf("INFO: Stopping connection maintenance due to context cancellation")
			return
		case <-pingTicker.C:
			// Safely access the connection
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				log.Printf("WARNING: Cannot send ping - WebSocket connection is nil")
				go c.reconnect()
				return
			}

			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Printf("ERROR: Ping error: %v", err)
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

	// Store current connection to close it after releasing the lock
	oldConn := c.conn
	// Set connection to nil to prevent other goroutines from using it
	c.conn = nil

	// Get current symbols
	var symbols []candlestick.Symbol
	for symbol := range c.handlers {
		symbols = append(symbols, symbol)
	}

	// Release the lock before attempting to reconnect
	c.mu.Unlock()

	// Close the old connection outside the lock to prevent deadlocks
	if oldConn != nil {
		// Ignore close errors as the connection might already be closed
		_ = oldConn.Close()
		log.Printf("INFO: Closed old WebSocket connection")
	}

	// Reacquire the lock for the reconnection attempt
	c.mu.Lock()

	// Attempt to reconnect
	if err := c.Connect(symbols); err != nil {
		log.Printf("ERROR: Reconnection failed: %v", err)
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
			// Check if connection is nil before attempting to read
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				log.Printf("ERROR: WebSocket connection is nil, waiting before reconnect attempt")
				time.Sleep(time.Second)
				go c.reconnect()
				return
			}

			_, message, err := conn.ReadMessage()
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

			var aggTradeMsg AggTradeMessage
			if err = json.Unmarshal(message, &aggTradeMsg); err != nil {
				parseErr := &MessageParsingError{MessageType: "aggTrade", RawMessage: string(message), Err: err}
				log.Printf("ERROR: %v", parseErr)
				continue
			}

			// Skip non-aggTrade messages
			if aggTradeMsg.EventType != "aggTrade" {
				log.Printf("DEBUG: Skipping non-aggTrade message type: %s", aggTradeMsg.EventType)
				continue
			}

			// Convert message to Tick with error handling
			symbol := candlestick.Symbol(aggTradeMsg.Symbol)
			price, err := strconv.ParseFloat(aggTradeMsg.Price, 64)
			if err != nil {
				log.Printf("ERROR: Failed parsing price for %s: %v", symbol, err)
				continue
			}

			quantity, err := strconv.ParseFloat(aggTradeMsg.Quantity, 64)
			if err != nil {
				log.Printf("ERROR: Failed parsing quantity for %s: %v", symbol, err)
				continue
			}

			tick := candlestick.Tick{
				Symbol:    symbol,
				Price:     price,
				Quantity:  quantity,
				Timestamp: time.Unix(0, aggTradeMsg.Timestamp*int64(time.Millisecond)),
			}

			log.Printf("INFO: Created tick for %s: price=%.2f quantity=%.2f timestamp=%s",
				symbol, price, quantity, tick.Timestamp.Format(time.RFC3339))

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
