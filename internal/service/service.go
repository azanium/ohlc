package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/azanium/ohlc/conf"
	"github.com/azanium/ohlc/internal/binance"
	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/azanium/ohlc/internal/storage"
	"github.com/azanium/ohlc/internal/streaming"
)

// Config holds service configuration
type Config struct {
	Symbols        []candlestick.Symbol
	Interval       time.Duration
	DBPath         string
	MaxSubscribers int
	ChannelSize    int
}

// Service coordinates the OHLC data processing pipeline
type Service struct {
	client     binance.BinanceClient
	aggregator candlestick.Aggregator
	storage    candlestick.Storage
	streamer   *streaming.Service
	config     Config
}

// SetClient allows injecting a mock client for testing
func (s *Service) SetClient(client binance.BinanceClient) {
	s.client = client
}

// New creates a new OHLC service
func New(ctx context.Context, config Config) (*Service, error) {
	// Create components
	client := binance.NewClient(ctx)
	aggregator := candlestick.NewAggregator(config.Interval)

	// Initialize storage with environment variables or defaults
	host := conf.GetConf().Postgres.Master.Address
	user := conf.GetConf().Postgres.Master.Username
	password := conf.GetConf().Postgres.Master.Password
	dbname := conf.GetConf().Postgres.Master.Database
	port := conf.GetConf().Postgres.Master.Port
	sslMode := conf.GetConf().Postgres.Master.SSLMode
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		host, user, password, dbname, port, sslMode)

	storage, err := storage.NewPostgreSQLStorage(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %v", err)
	}

	// Initialize streaming service
	streamer := streaming.NewService(config.MaxSubscribers, config.ChannelSize)

	return &Service{
		client:     client,
		aggregator: aggregator,
		storage:    storage,
		streamer:   streamer,
		config:     config,
	}, nil
}

// Start begins the OHLC data processing
func (s *Service) Start(ctx context.Context) error {
	// Connect to Binance
	if err := s.client.Connect(s.config.Symbols); err != nil {
		return fmt.Errorf("failed to connect to Binance: %v", err)
	}

	// Create tick channel
	tickCh := make(chan candlestick.Tick, 1000)

	// Subscribe to ticks
	for _, symbol := range s.config.Symbols {
		s.client.Subscribe(symbol, tickCh)
	}

	// Process ticks
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case tick := <-tickCh:
				// Process tick and get completed OHLC if available
				ohlc, err := s.aggregator.Process(tick)
				if err != nil {
					log.Printf("Error processing tick: %v", err)
					continue
				}

				// If we have a completed OHLC, store and stream it
				if ohlc != nil {
					// Store OHLC
					if err := s.storage.Store(ohlc); err != nil {
						log.Printf("Error storing OHLC: %v", err)
					}

					// Stream OHLC
					if err := s.streamer.Stream(ohlc); err != nil {
						log.Printf("Error streaming OHLC: %v", err)
					}
				}
			}
		}
	}()

	return nil
}

// Stop gracefully shuts down the service
func (s *Service) Stop() error {
	// Close Binance connection
	if err := s.client.Close(); err != nil {
		log.Printf("Error closing Binance client: %v", err)
	}

	// Close storage
	if err := s.storage.(*storage.PostgreSQLStorage).Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	return nil
}

// GetStreamer returns the gRPC streaming service
func (s *Service) GetStreamer() *streaming.Service {
	return s.streamer
}
