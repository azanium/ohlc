package streaming

import (
	"sync"

	"github.com/azanium/ohlc/internal/candlestick"
	"github.com/azanium/ohlc/internal/proto/proto"
)

// Service implements the gRPC streaming service
type Service struct {
	proto.UnimplementedOHLCServiceServer
	mu          sync.RWMutex
	subscribers map[candlestick.Symbol][]chan *candlestick.OHLC
	maxChannels int
	channelSize int
}

// NewService creates a new streaming service
func NewService(maxChannels, channelSize int) *Service {
	return &Service{
		subscribers: make(map[candlestick.Symbol][]chan *candlestick.OHLC),
		maxChannels: maxChannels,
		channelSize: channelSize,
	}
}

// StreamOHLC implements the gRPC streaming endpoint
func (s *Service) StreamOHLC(req *proto.SubscribeRequest, stream proto.OHLCService_StreamOHLCServer) error {
	// Create channels for each requested symbol
	channels := make(map[candlestick.Symbol]chan *candlestick.OHLC)
	for _, symbolStr := range req.Symbols {
		symbol := candlestick.Symbol(symbolStr)
		ch := make(chan *candlestick.OHLC, s.channelSize)
		channels[symbol] = ch

		// Subscribe to updates
		s.Subscribe(symbol, ch)
	}

	// Clean up on exit
	defer func() {
		for symbol, ch := range channels {
			s.unsubscribe(symbol, ch)
			close(ch)
		}
	}()

	// Stream updates to client
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			// Receive from any channel
			for _, ch := range channels {
				select {
				case ohlc := <-ch:
					// Convert to proto message
					msg := &proto.OHLCData{
						Symbol:    string(ohlc.Symbol),
						Open:      ohlc.Open,
						High:      ohlc.High,
						Low:       ohlc.Low,
						Close:     ohlc.Close,
						Volume:    ohlc.Volume,
						OpenTime:  ohlc.OpenTime.UnixMilli(),
						CloseTime: ohlc.CloseTime.UnixMilli(),
					}

					if err := stream.Send(msg); err != nil {
						return err
					}
				default:
					continue
				}
			}
		}
	}
}

// Stream broadcasts an OHLC update to all subscribers
func (s *Service) Stream(ohlc *candlestick.OHLC) error {
	s.mu.RLock()
	subscribers := s.subscribers[ohlc.Symbol]
	s.mu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- ohlc:
		default:
			// Skip if channel is full
		}
	}

	return nil
}

// subscribe adds a subscriber channel for a symbol
func (s *Service) Subscribe(symbol candlestick.Symbol, ch chan *candlestick.OHLC) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers[symbol] = append(s.subscribers[symbol], ch)
}

// unsubscribe removes a subscriber channel
func (s *Service) unsubscribe(symbol candlestick.Symbol, ch chan *candlestick.OHLC) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[symbol]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[symbol] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}
