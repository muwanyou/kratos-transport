package websocket

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/muwanyou/kratos-transport/enum"
)

var channelSize = 256

type Session struct {
	id     string
	server *Server
	conn   *websocket.Conn
	send   chan []byte

	lastReadMessageTime  time.Time
	lastWriteMessageTime time.Time
}

func NewSession(server *Server, conn *websocket.Conn) *Session {
	if conn == nil {
		panic("conn cannot be nil")
	}
	session := &Session{
		id:     uuid.NewString(),
		server: server,
		conn:   conn,
		send:   make(chan []byte, channelSize),
	}
	return session
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Listen() {
	go s.read()
	go s.write()
}

func (s *Session) Close() {
	s.server.unregister <- s
	s.closeConnect()
}

func (s *Session) closeConnect() {
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			log.Errorf("disconnect error: %v", err)
		}
		s.conn = nil
	}
}

func (s *Session) sendBinaryMessage(message []byte) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, message)
}

func (s *Session) sendTextMessage(message string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (s *Session) sendPingMessage(message string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.WriteMessage(websocket.PingMessage, []byte(message))
}

func (s *Session) write() {
	defer s.Close()
	for {
		select {
		case bytes := <-s.send:
			s.lastWriteMessageTime = time.Now()
			var err error
			switch s.server.messageType {
			case enum.MessageTypeBinary:
				if err = s.sendBinaryMessage(bytes); err != nil {
					log.Errorf("write binary message error: %v", err)
					return
				}
				break
			case enum.MessageTypeText:
				if err = s.sendTextMessage(string(bytes)); err != nil {
					log.Errorf("write text message error: %v", err)
					return
				}
				break
			}
		}
	}
}

func (s *Session) read() {
	defer s.Close()
	for {
		if s.conn == nil {
			break
		}
		messageType, data, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("read message error: %v", err)
			}
			return
		}
		s.lastReadMessageTime = time.Now()
		switch messageType {
		case websocket.CloseMessage:
			return
		case websocket.BinaryMessage:
			_ = s.server.readHandler(s.ID(), data)
			break
		case websocket.TextMessage:
			_ = s.server.readHandler(s.ID(), data)
			break
		case websocket.PingMessage:
			if err = s.sendPingMessage(""); err != nil {
				log.Errorf("write ping message error: %v", err)
				return
			}
			break
		case websocket.PongMessage:
			break
		}
	}
}
