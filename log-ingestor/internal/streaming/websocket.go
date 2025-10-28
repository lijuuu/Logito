package streaming

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type StreamServer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
}

func NewStreamServer() *StreamServer {
	return &StreamServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

func (s *StreamServer) HandleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("Failed to upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	logger.Info("WebSocket client connected")

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		logger.Info("WebSocket client disconnected")
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (s *StreamServer) BroadcastBatch(batch *logentry.Batch) {
	if batch.IsEmpty() {
		return
	}

	streamData := struct {
		Timestamp int64                `json:"timestamp"`
		Entries   []*logentry.LogEntry `json:"entries"`
	}{
		Timestamp: time.Now().UnixMilli(),
		Entries:   batch.Entries,
	}

	data, err := json.Marshal(streamData)
	if err != nil {
		logger.Error("Failed to marshal batch: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for conn := range s.clients {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logger.Error("Failed to send batch to client: %v", err)
			conn.Close()
			delete(s.clients, conn)
		}
	}
}

func (s *StreamServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
