package bot

import "context"

// EventLoop serializes access to shared state by processing commands
// on a single goroutine, replicating Node.js's single-threaded model.
type EventLoop struct {
	cmds chan func()
}

// NewEventLoop creates an event loop with a buffered command channel.
func NewEventLoop() *EventLoop {
	return &EventLoop{cmds: make(chan func(), 64)}
}

// Run processes commands until the context is cancelled.
func (l *EventLoop) Run(ctx context.Context) {
	for {
		select {
		case fn := <-l.cmds:
			fn()
		case <-ctx.Done():
			return
		}
	}
}

// Do sends a synchronous command and blocks until it completes.
func (l *EventLoop) Do(fn func()) {
	done := make(chan struct{})
	l.cmds <- func() {
		fn()
		close(done)
	}
	<-done
}
