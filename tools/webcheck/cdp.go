package main

// A client for the Chrome DevTools Protocol over --remote-debugging-pipe.
//
// The pipe rather than the usual websocket because the standard library has no websocket, and the
// pipe needs nothing: Chrome reads commands from file descriptor 3 and writes replies and events
// to 4, each message one JSON value ending in a NUL byte. exec.Cmd.ExtraFiles puts two pipe ends
// at exactly those numbers.
//
// It is synchronous. A command is written, and messages are read until its reply arrives; events
// that arrive first are kept for whoever waits for them. There is no reader goroutine, so there is
// nothing to race and nothing to stop, and a browser that has hung is a read deadline running out
// rather than a goroutine blocked for ever.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// message is every shape the protocol has: a command, a reply and an event.
type message struct {
	ID        int             `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// conn is one side of the pipe pair. The deadline-setting half is an interface so that the tests
// can give it a pipe of their own.
type conn struct {
	w  io.Writer
	r  io.Reader
	rd interface{ SetReadDeadline(time.Time) error }

	// pending is what has been read and is not yet a whole message. It outlives a read that
	// timed out, which is the reason it is here and not inside a bufio.Reader: a deadline that
	// runs out halfway through a message must not cost the next read its first half.
	pending []byte
	buf     []byte

	next   int
	events []message
}

func newConn(w io.Writer, r io.Reader, rd interface{ SetReadDeadline(time.Time) error }) *conn {
	return &conn{w: w, r: r, rd: rd, buf: make([]byte, 1<<16)}
}

// send writes one command and returns its id.
func (c *conn) send(session, method string, params any) (int, error) {
	c.next++
	m := struct {
		ID        int    `json:"id"`
		Method    string `json:"method"`
		Params    any    `json:"params"`
		SessionID string `json:"sessionId,omitempty"`
	}{c.next, method, params, session}
	if m.Params == nil {
		m.Params = struct{}{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	if _, err := c.w.Write(append(b, 0)); err != nil {
		return 0, fmt.Errorf("%s: %w", method, err)
	}
	return c.next, nil
}

// read is the next message, or an error once the deadline has passed.
func (c *conn) read(deadline time.Time) (message, error) {
	for {
		if i := bytes.IndexByte(c.pending, 0); i >= 0 {
			raw := c.pending[:i]
			c.pending = c.pending[i+1:]
			var m message
			if err := json.Unmarshal(raw, &m); err != nil {
				return message{}, fmt.Errorf("the browser sent something that is not JSON: %w", err)
			}
			return m, nil
		}
		if err := c.rd.SetReadDeadline(deadline); err != nil {
			return message{}, err
		}
		n, err := c.r.Read(c.buf)
		c.pending = append(c.pending, c.buf[:n]...)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return message{}, errTimeout
			}
			return message{}, fmt.Errorf("reading from the browser: %w", err)
		}
	}
}

var errTimeout = errors.New("timed out")

// call sends a command and waits for its reply, keeping the events that arrive first.
func (c *conn) call(session, method string, params any, timeout time.Duration) (json.RawMessage, error) {
	id, err := c.send(session, method, params)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		m, err := c.read(deadline)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", method, err)
		}
		if m.ID == 0 {
			c.events = append(c.events, m)
			continue
		}
		if m.ID != id {
			continue // the reply to a command whose caller already gave up on it
		}
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	}
}

// event waits for one event of this method on this session, kept or yet to come, and takes it.
func (c *conn) event(session, method string, timeout time.Duration) (message, error) {
	deadline := time.Now().Add(timeout)
	for {
		for i, m := range c.events {
			if m.Method == method && m.SessionID == session {
				c.events = append(c.events[:i], c.events[i+1:]...)
				return m, nil
			}
		}
		m, err := c.read(deadline)
		if err != nil {
			return message{}, fmt.Errorf("waiting for %s: %w", method, err)
		}
		if m.ID == 0 {
			c.events = append(c.events, m)
		}
	}
}

// drop forgets the events kept for a session, other than the ones worth reporting.
func (c *conn) drop(session string, keep func(message) bool) []message {
	var kept, rest []message
	for _, m := range c.events {
		switch {
		case m.SessionID != session:
			rest = append(rest, m)
		case keep(m):
			kept = append(kept, m)
		}
	}
	c.events = rest
	return kept
}
