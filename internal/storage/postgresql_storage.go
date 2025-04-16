package storage

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/azanium/ohlc/internal/candlestick"
)

// Custom error types for better error handling
type StorageError struct {
	Operation string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage operation '%s' failed: %v", e.Operation, e.Err)
}

type QueryError struct {
	Symbol candlestick.Symbol
	Start  time.Time
	End    time.Time
	Err    error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("query for symbol %s from %v to %v failed: %v", e.Symbol, e.Start, e.End, e.Err)
}

// PostgreSQLStorage implements the candlestick.Storage interface using PostgreSQL
type PostgreSQLStorage struct {
	db *gorm.DB
}

// Store persists an OHLC candlestick
func (s *PostgreSQLStorage) Store(ohlc *candlestick.OHLC) error {
	log.Printf("Storing OHLC: symbol=%s, open=%.2f, high=%.2f, low=%.2f, close=%.2f, volume=%.2f, openTime=%s, closeTime=%s",
		ohlc.Symbol, ohlc.Open, ohlc.High, ohlc.Low, ohlc.Close, ohlc.Volume,
		ohlc.OpenTime.Format(time.RFC3339), ohlc.CloseTime.Format(time.RFC3339))

	err := s.db.Create(ohlc).Error
	if err != nil {
		storageErr := &StorageError{Operation: "store_ohlc", Err: err}
		log.Printf("Error: %v", storageErr)
		return storageErr
	}
	return nil
}

// GetRange retrieves OHLC candlesticks for a symbol within a time range
func (s *PostgreSQLStorage) GetRange(symbol candlestick.Symbol, start, end time.Time) ([]*candlestick.OHLC, error) {
	var result []*candlestick.OHLC
	err := s.db.Where("symbol = ? AND open_time >= ? AND close_time <= ?", symbol, start.UnixMilli(), end.UnixMilli()).Order("open_time ASC").Find(&result).Error
	if err != nil {
		queryErr := &QueryError{Symbol: symbol, Start: start, End: end, Err: err}
		log.Printf("Error: %v", queryErr)
		return nil, queryErr
	}
	return result, nil
}

// StoreTick persists a tick to the database
func (s *PostgreSQLStorage) StoreTick(tick *candlestick.Tick) error {
	log.Printf("Storing tick: symbol=%s, price=%.2f, quantity=%.2f, timestamp=%s",
		tick.Symbol, tick.Price, tick.Quantity, tick.Timestamp.Format(time.RFC3339))

	err := s.db.Model(&candlestick.Tick{}).Create(tick).Error
	if err != nil {
		storageErr := &StorageError{Operation: "store_tick", Err: err}
		log.Printf("Error: %v", storageErr)
		return storageErr
	}
	return nil
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewPostgreSQLStorage(dsn string) (*PostgreSQLStorage, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Auto migrate the schema
	db.AutoMigrate(&candlestick.OHLC{}, &candlestick.Tick{})

	return &PostgreSQLStorage{db: db}, nil
}

// Close closes the database connection
func (s *PostgreSQLStorage) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
