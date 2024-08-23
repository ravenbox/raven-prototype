package utils

import "sync"

type Mutex struct{ sync.Mutex }

func (m *Mutex) Tx(fn func()) {
	m.Lock()
	defer m.Unlock()
	fn()
}

type RWMutex struct{ sync.RWMutex }

func (m *RWMutex) Tx(fn func()) {
	m.Lock()
	defer m.Unlock()
	fn()
}

func (m *RWMutex) Rx(fn func()) {
	m.RLock()
	defer m.RUnlock()
	fn()
}
