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

type failGroup struct {
	failed  chan struct{}
	failNow func()
	wg      sync.WaitGroup
}

// A FailGroup waits for a collection of goroutines to either finish or fail.
func FailGroup() *failGroup {
	fg := &failGroup{
		failed: make(chan struct{}),
	}
	fg.failNow = sync.OnceFunc(func() {
		close(fg.failed)
	})
	return fg
}

// Wait returns a channel which returns true when all Checks done successfully.
func (fg *failGroup) Wait() <-chan bool {
	success := make(chan bool)
	go func() {
		fg.wg.Wait()
		select {
		case <-fg.failed:
			success <- false
		default:
			success <- true
		}
	}()
	return success
}

// Check provides a fail and a done callback for task goroutine.
func (fg *failGroup) Check(fn func(fail, done func())) {
	fg.wg.Add(1)
	ch := make(chan struct{})
	fail := func() {
		ch <- struct{}{}
	}
	done := func() {
		close(ch)
	}
	go func() {
		defer fg.wg.Done()
		for range ch {
			fg.failNow()
			return
		}
	}()
	fn(fail, done)
}

// AsyncCheck only provides fail callback and it does not block Wait.
func (fg *failGroup) AsyncCheck(fn func(fail func())) {
	ch := make(chan struct{})
	fail := func() {
		close(ch)
	}
	go func() {
		<-ch
		fg.failNow()
	}()
	fn(fail)
}
