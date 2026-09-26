// Package upstreamtiming observes transport events without logging request data.
package upstreamtiming

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

type Event struct {
	Name   string `json:"name"`
	AtMS   int64  `json:"at_ms"`
	Failed bool   `json:"failed,omitempty"`
	Reused bool   `json:"reused,omitempty"`
}

type Snapshot struct {
	Stage         string  `json:"stage"`
	StartedUnixMS int64   `json:"started_unix_ms"`
	Events        []Event `json:"events"`
	Dropped       int     `json:"dropped_events"`
	Status        int     `json:"status"`
	Protocol      string  `json:"protocol"`
}

type Trace struct {
	mu        sync.Mutex
	start     time.Time
	events    []Event
	dropped   int
	status    int
	protocol  string
	firstBody sync.Once
	firstSSE  sync.Once
	finished  sync.Once
	emit      func(Snapshot)
}

func New(emit func(Snapshot)) *Trace {
	return &Trace{start: time.Now(), emit: emit}
}

func (t *Trace) record(name string, failed, reused bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Keep repeated dial attempts and HTTP/2 fallback visible, with bounded memory.
	if len(t.events) >= 128 {
		t.dropped++
		return
	}
	t.events = append(t.events, Event{name, time.Since(t.start).Milliseconds(), failed, reused})
}

func (t *Trace) snapshot(stage string) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return Snapshot{stage, t.start.UnixMilli(), append([]Event(nil), t.events...), t.dropped, t.status, t.protocol}
}

func (t *Trace) publish(stage string) {
	if t.emit != nil {
		t.emit(t.snapshot(stage))
	}
}

func (t *Trace) Context(ctx context.Context) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn:              func(string) { t.record("get_conn", false, false) },
		GotConn:              func(i httptrace.GotConnInfo) { t.record("got_conn", false, i.Reused) },
		DNSStart:             func(httptrace.DNSStartInfo) { t.record("dns_start", false, false) },
		DNSDone:              func(i httptrace.DNSDoneInfo) { t.record("dns_done", i.Err != nil, false) },
		ConnectStart:         func(string, string) { t.record("connect_start", false, false) },
		ConnectDone:          func(_, _ string, err error) { t.record("connect_done", err != nil, false) },
		TLSHandshakeStart:    func() { t.record("tls_start", false, false) },
		TLSHandshakeDone:     func(_ tls.ConnectionState, err error) { t.record("tls_done", err != nil, false) },
		WroteHeaders:         func() { t.record("wrote_headers", false, false) },
		WroteRequest:         func(i httptrace.WroteRequestInfo) { t.record("wrote_request", i.Err != nil, false) },
		GotFirstResponseByte: func() { t.record("first_response_byte", false, false) },
	})
}

func (t *Trace) Wrap(resp *http.Response) {
	t.mu.Lock()
	t.status, t.protocol = resp.StatusCode, resp.Proto
	t.mu.Unlock()
	t.record("headers_ready", false, false)
	resp.Body = &body{ReadCloser: resp.Body, trace: t}
}

// FirstSSE must be called by the existing parser, using its existing TTFT rule.
func (t *Trace) FirstSSE() {
	t.firstSSE.Do(func() {
		t.record("first_sse", false, false)
		t.publish("first_sse")
	})
}

func (t *Trace) Finish(failed bool) {
	t.finished.Do(func() {
		t.record("finished", failed, false)
		t.publish("finished")
	})
}

type body struct {
	io.ReadCloser
	trace *Trace
}

func (b *body) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.trace.firstBody.Do(func() { b.trace.record("first_body_read", false, false) })
	}
	if err != nil && err != io.EOF {
		b.trace.record("body_read_error", true, false)
	}
	return n, err
}

func (b *body) Close() error {
	err := b.ReadCloser.Close()
	b.trace.Finish(err != nil)
	return err
}
