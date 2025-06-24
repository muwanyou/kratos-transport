package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"github.com/muwanyou/kratos-websocket/enum"
	"github.com/muwanyou/kratos-websocket/util"
)

type Server struct {
	*http.Server
	listener    net.Listener
	tlsConfig   *tls.Config
	endpoint    *url.URL
	err         error
	network     string
	address     string
	path        string
	upgrader    *websocket.Upgrader
	manager     *Manager
	register    chan *Session
	unregister  chan *Session
	messageType enum.MessageType
	readHandler ReadHandler
}

type ServerOption func(*Server)

type ReadHandler func(sessionID string, bytes []byte) error

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network: "tcp",
		address: ":0",
		path:    "/",
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		manager:     NewManager(),
		register:    make(chan *Session),
		unregister:  make(chan *Session),
		readHandler: defaultReadHandler,
	}
	for _, opt := range opts {
		opt(srv)
	}
	srv.Server = &http.Server{
		TLSConfig: srv.tlsConfig,
	}
	http.HandleFunc(srv.path, srv.handler)
	return srv
}

func Network(network string) ServerOption {
	return func(s *Server) {
		if network != "" {
			s.network = network
		}
	}
}

func Address(address string) ServerOption {
	return func(s *Server) {
		if address != "" {
			s.address = address
		}
	}
}

func Path(path string) ServerOption {
	return func(s *Server) {
		if path != "" {
			s.path = path
		}
	}
}

func TLSConfig(config *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConfig = config
	}
}

func ReadBufferSize(size int) ServerOption {
	return func(s *Server) {
		s.upgrader.ReadBufferSize = size
	}
}

func WriteBufferSize(size int) ServerOption {
	return func(s *Server) {
		s.upgrader.WriteBufferSize = size
	}
}

func MessageType(messageType enum.MessageType) ServerOption {
	return func(s *Server) {
		s.messageType = messageType
	}
}

func RegisterReadHandler(srv *Server, handler ReadHandler) {
	srv.readHandler = handler
}

func defaultReadHandler(sessionID string, bytes []byte) error {
	return nil
}

func (s *Server) run() {
	for {
		select {
		case session := <-s.register:
			s.manager.Add(session)
		case session := <-s.unregister:
			s.manager.Remove(session)
		}
	}
}

func (s *Server) handler(res http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Errorf("upgrade exception: %v", err)
	}
	session := NewSession(s, conn)
	s.register <- session
	session.Listen()
}

func (s *Server) listenAndEndpoint() error {
	if s.listener == nil {
		listener, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return err
		}
		s.listener = listener
	}
	if s.endpoint == nil {
		addr, err := util.Extract(s.address, s.listener)
		if err != nil {
			s.err = err
			return err
		}
		s.endpoint = util.NewEndpoint(util.Scheme("http", s.tlsConfig != nil), addr)
	}
	return s.err
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	log.Infof("[Websocket] server listening on: %s", s.listener.Addr().String())
	go s.run()
	var err error
	if s.tlsConfig != nil {
		err = s.ServeTLS(s.listener, "", "")
	} else {
		err = s.Serve(s.listener)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[Websocket] server stopping")
	err := s.Shutdown(ctx)
	if err != nil {
		if ctx.Err() != nil {
			log.Warn("[Websocket] server couldn't stop gracefully in time, doing force stop")
			err = s.Server.Close()
			return err
		}
		return err
	}
	return nil
}
