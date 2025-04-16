package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azanium/ohlc/conf"
	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/azanium/ohlc/internal/service"
	"github.com/azanium/ohlc/proto"
	"google.golang.org/grpc"
)

func main() {
	// Create a context with cancellation for coordinated shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Initialize storage with environment variables or defaults
	host := conf.GetConf().Postgres.Master.Address
	user := conf.GetConf().Postgres.Master.Username
	password := conf.GetConf().Postgres.Master.Password
	dbname := conf.GetConf().Postgres.Master.Database
	port := conf.GetConf().Postgres.Master.Port
	sslMode := conf.GetConf().Postgres.Master.SSLMode
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		host, user, password, dbname, port, sslMode)
	// Service configuration
	config := service.Config{
		Symbols: []candlestick.Symbol{
			candlestick.BTCUSDT,
			candlestick.ETHUSDT,
			candlestick.PEPEUSDT,
		},
		Interval:       10 * time.Second,
		StorageDSN:     dsn,
		MaxSubscribers: 100,
		ChannelSize:    1000,
	}

	log.Printf("Starting OHLC service with configuration: %+v", config)

	// Create and start service
	svc, err := service.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	if err = svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", conf.GetConf().Server.Address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterOHLCServiceServer(grpcServer, svc.GetStreamer())

	go func() {
		log.Printf("Starting gRPC server on %s", conf.GetConf().Server.Address)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigCh
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Cancel the context to notify all components
	cancel()

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Channel to track shutdown completion
	shutdownComplete := make(chan struct{})

	go func() {
		// Stop accepting new gRPC requests
		log.Println("Stopping gRPC server...")
		grpcServer.GracefulStop()
		log.Println("gRPC server stopped")

		// Stop the service
		log.Println("Stopping OHLC service...")
		if err := svc.Stop(); err != nil {
			log.Printf("Error during service shutdown: %v", err)
		}

		close(shutdownComplete)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		log.Println("Service shutdown completed successfully")
	case <-shutdownCtx.Done():
		log.Println("Service shutdown timed out")
	}

}
