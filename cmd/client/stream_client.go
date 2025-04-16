package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/azanium/ohlc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Starting client...")

	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a gRPC client
	client := proto.NewOHLCServiceClient(conn)

	// Create a subscription request for BTC and ETH
	req := &proto.SubscribeRequest{
		Symbols: []string{"BTCUSDT", "ETHUSDT", "PEPEUSDT"},
	}

	// Start streaming OHLC data
	ctx := context.Background()
	stream, err := client.StreamOHLC(ctx, req)
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}

	// Track last OHLC time for each symbol
	lastOHLCTime := make(map[string]time.Time)

	// Receive and print streaming updates
	for {
		ohlc, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error receiving: %v", err)
		}

		// Print the received OHLC data
		openTime := time.UnixMilli(ohlc.OpenTime)
		closeTime := time.UnixMilli(ohlc.CloseTime)
		interval := ""
		if lastTime, ok := lastOHLCTime[ohlc.Symbol]; ok {
			interval = fmt.Sprintf(" (Interval: %v)", openTime.Sub(lastTime).Round(time.Second))
		}
		lastOHLCTime[ohlc.Symbol] = openTime

		fmt.Printf("[%s] %s - Open: %.2f, High: %.2f, Low: %.2f, Close: %.2f, Volume: %.2f (Period: %s - %s)%s\n",
			ohlc.Symbol,
			openTime.Format("15:04:05"),
			ohlc.Open,
			ohlc.High,
			ohlc.Low,
			ohlc.Close,
			ohlc.Volume,
			openTime.Format("15:04:05"),
			closeTime.Format("15:04:05"),
			interval,
		)
	}
}
