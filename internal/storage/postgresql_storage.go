package storage

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/azanium/ohlc/internal/candlestick"
)

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
		log.Printf("Error storing OHLC: %v", err)
		return err
	}
	return nil
}

// GetRange retrieves OHLC candlesticks for a symbol within a time range
func (s *PostgreSQLStorage) GetRange(symbol candlestick.Symbol, start, end time.Time) ([]*candlestick.OHLC, error) {
	var result []*candlestick.OHLC
	err := s.db.Where("symbol = ? AND open_time >= ? AND close_time <= ?", symbol, start.UnixMilli(), end.UnixMilli()).Order("open_time ASC").Find(&result).Error
	return result, err
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewPostgreSQLStorage(dsn string) (*PostgreSQLStorage, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Auto migrate the schema
	db.AutoMigrate(&candlestick.OHLC{})

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
