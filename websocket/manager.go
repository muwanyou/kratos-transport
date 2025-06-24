package websocket

import (
	"sync"
)

type Manager struct {
	mutex      sync.RWMutex
	sessionMap map[string]*Session
}

func NewManager() *Manager {
	return &Manager{
		sessionMap: make(map[string]*Session),
	}
}

func (m *Manager) Get(sessionID string) (*Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	session, ok := m.sessionMap[sessionID]
	return session, ok
}

func (m *Manager) Add(session *Session) {
	m.mutex.Lock()
	m.sessionMap[session.ID()] = session
	m.mutex.Unlock()
}

func (m *Manager) Remove(session *Session) {
	m.mutex.Lock()
	if _, ok := m.sessionMap[session.ID()]; ok {
		delete(m.sessionMap, session.ID())
	}
	m.mutex.Unlock()
}

func (m *Manager) Count() int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return int64(len(m.sessionMap))
}

func (m *Manager) Clean() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sessionMap = map[string]*Session{}
}
