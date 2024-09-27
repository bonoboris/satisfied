// A custom colored [slog.Handler] inspired by [tint package](https://github.com/lmittmann/tint)
// but simplified.

package log

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"strconv"
)

// ANSI modes
const (
	ansiReset          = "\033[0m"
	ansiFaint          = "\033[2m"
	ansiResetFaint     = "\033[22m"
	ansiBrightRed      = "\033[91m"
	ansiBrightGreen    = "\033[92m"
	ansiBrightYellow   = "\033[93m"
	ansiBrightBlue     = "\033[94m"
	ansiBrightRedFaint = "\033[91;2m"
)

const (
	lowerLvlStrNC  = "TRC-"
	traceLvlStrNC  = "TRC "
	debugLvlStrNC  = "DBG "
	infoLvlStrNC   = "INF "
	warnLvlStrNC   = "WRN "
	errorLvlStrNC  = "ERR "
	fatalLvlStrNC  = "FTL "
	higherLvlStrNC = "FTL+"
)

const (
	lowerLvlStr  = "TRC-"
	traceLvlStr  = "TRC "
	debugLvlStr  = ansiBrightBlue + "DBG " + ansiReset
	infoLvlStr   = ansiBrightGreen + "INF " + ansiReset
	warnLvlStr   = ansiBrightYellow + "WRN " + ansiReset
	errorLvlStr  = ansiBrightRed + "ERR " + ansiReset
	fatalLvlStr  = ansiBrightRed + "FTL " + ansiReset
	higherLvlStr = "FTL+"
)

type handler struct {
	ch      chan slog.Record
	stopCh  chan struct{}
	buffer  []byte
	writer  io.Writer
	level   slog.Level
	colored bool
}

func NewHandler(writer io.Writer, level slog.Level, colored bool) *handler {
	ch := make(chan slog.Record, 1024)
	stopCh := make(chan struct{})
	buffer := make([]byte, 0, 1024)
	return &handler{ch, stopCh, buffer, writer, level, colored}
}

// Starts the handler goroutine
func (h *handler) Start() { go h.run() }

// Stops the handler goroutine
func (h *handler) Stop() { h.stopCh <- struct{}{} }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	h.ch <- r
	return nil
}

func (h *handler) run() {
	for {
		select {
		case r := <-h.ch:
			h.doHandle(r)
		case <-h.stopCh:
			return
		}
	}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	panic("not implemented")
}

func (h *handler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *handler) padBuffer(n int) {
	for range n - len(h.buffer) {
		h.buffer = append(h.buffer, ' ')
	}
}

func (h *handler) doHandle(r slog.Record) error {
	h.buffer = h.buffer[:0]
	nSource := h.writeSource(r.PC)
	// pad source to 30 chars
	h.padBuffer(24 + len(h.buffer) - nSource)
	h.writeLevel(r.Level)
	h.buffer = append(h.buffer, ' ')
	h.buffer = append(h.buffer, r.Message...)
	h.padBuffer(50 + len(h.buffer) - 5 - len(r.Message))
	h.writeAttrs(r)
	h.buffer = append(h.buffer, '\n')
	_, err := h.writer.Write(h.buffer)
	return err
}

// Writes the log source to the buffer and returns the length of the source (excluding ansi codes)
func (h *handler) writeSource(pc uintptr) int {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, more := fs.Next()
	if more {
		f, _ = fs.Next()
	}
	if f.File == "" {
		return 0
	}

	// find the idx of the last directory start
	idx := 0
	firstFound := false
	for i := len(f.File) - 1; i >= 0; i-- {
		if f.File[i] == '/' {
			if !firstFound {
				firstFound = true
			} else {
				idx = i + 1
				break
			}
		}
	}
	sLine := strconv.Itoa(f.Line)
	if h.colored {
		h.buffer = append(h.buffer, ansiFaint...)
		h.buffer = append(h.buffer, f.File[idx:]...)
		h.buffer = append(h.buffer, ':')
		h.buffer = append(h.buffer, sLine...)
		h.buffer = append(h.buffer, ansiReset...)
	} else {
		h.buffer = append(h.buffer, f.File[idx:]...)
		h.buffer = append(h.buffer, ':')
		h.buffer = append(h.buffer, sLine...)
	}
	return len(f.File) - idx + 1 + len(sLine)
}

func (h *handler) writeLevel(lvl slog.Level) {
	if h.colored {
		switch lvl {
		case TraceLevel:
			h.buffer = append(h.buffer, traceLvlStr...)
		case DebugLevel:
			h.buffer = append(h.buffer, debugLvlStr...)
		case InfoLevel:
			h.buffer = append(h.buffer, infoLvlStr...)
		case WarnLevel:
			h.buffer = append(h.buffer, warnLvlStr...)
		case ErrorLevel:
			h.buffer = append(h.buffer, errorLvlStr...)
		case FatalLevel:
			h.buffer = append(h.buffer, fatalLvlStr...)
		default:
			if lvl < TraceLevel {
				h.buffer = append(h.buffer, lowerLvlStr...)
			} else {
				h.buffer = append(h.buffer, higherLvlStr...)
			}
		}
	} else {
		switch lvl {
		case TraceLevel:
			h.buffer = append(h.buffer, traceLvlStrNC...)
		case DebugLevel:
			h.buffer = append(h.buffer, debugLvlStrNC...)
		case InfoLevel:
			h.buffer = append(h.buffer, infoLvlStrNC...)
		case WarnLevel:
			h.buffer = append(h.buffer, warnLvlStrNC...)
		case ErrorLevel:
			h.buffer = append(h.buffer, errorLvlStrNC...)
		case FatalLevel:
			h.buffer = append(h.buffer, fatalLvlStrNC...)
		default:
			if lvl < TraceLevel {
				h.buffer = append(h.buffer, lowerLvlStrNC...)
			} else {
				h.buffer = append(h.buffer, higherLvlStrNC...)
			}
		}
	}
}

func (h *handler) writeAttrs(r slog.Record) {
	if h.colored {
		r.Attrs(func(attr slog.Attr) bool {
			if attr.Value.Kind() == slog.KindString {
				attr.Value = slog.StringValue(strconv.Quote(attr.Value.String()))
			}
			h.buffer = append(h.buffer, ' ')
			h.buffer = append(h.buffer, ansiFaint...)
			h.buffer = append(h.buffer, attr.Key...)
			h.buffer = append(h.buffer, ansiReset...)
			h.buffer = append(h.buffer, '=')
			h.buffer = append(h.buffer, attr.Value.String()...)
			return true
		})
	} else {
		r.Attrs(func(attr slog.Attr) bool {
			if attr.Value.Kind() == slog.KindString {
				attr.Value = slog.StringValue(strconv.Quote(attr.Value.String()))
			}
			h.buffer = append(h.buffer, ' ')
			h.buffer = append(h.buffer, attr.Key...)
			h.buffer = append(h.buffer, '=')
			h.buffer = append(h.buffer, attr.Value.String()...)
			return true
		})
	}
}
