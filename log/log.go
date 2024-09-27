package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	AllLevel slog.Level = iota
	TraceLevel
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

type Options struct {
	// Terminal (stderr) log level (All log by default)
	Level slog.Level
	// Log to file (empty string to disable)
	FileWriter io.Writer
	// Log file level (All log by default)
	FileLevel slog.Level
	// Whether to colorize log output in terminal
	Colored bool
}

var (
	termHandler *handler
	fileHandler *handler
	minLevel    slog.Level
)

// Initializes logging
func Init(opts Options) {
	minLevel = min(opts.Level, opts.FileLevel)
	slog.SetLogLoggerLevel(opts.Level)
	rl.SetTraceLogLevel(rl.TraceLogLevel(opts.Level))
	rl.SetTraceLogCallback(func(l int, message string) { Log(slog.Level(l), message, "source", "raylib") })
	termHandler = NewHandler(os.Stderr, opts.Level, opts.Colored)
	termHandler.Start()
	if opts.FileWriter != nil {
		fileHandler = NewHandler(opts.FileWriter, opts.FileLevel, false)
		fileHandler.Start()
	}
}

// Close stops the log handling goroutines
func Close() {
	if termHandler != nil {
		termHandler.Stop()
	}
	if fileHandler != nil {
		fileHandler.Stop()
	}
}

// WillTrace returns true if [TraceLevel] logs will be written
func WillTrace() bool { return minLevel <= TraceLevel }

// Log at [TraceLevel]
func Trace(msg string, args ...any) { log(TraceLevel, msg, args...) }

// Log at [DebugLevel]
func Debug(msg string, args ...any) { log(DebugLevel, msg, args...) }

// Log at [InfoLevel]
func Info(msg string, args ...any) { log(InfoLevel, msg, args...) }

// Log at [WarnLevel]
func Warn(msg string, args ...any) { log(WarnLevel, msg, args...) }

// Log at [ErrorLevel]
func Error(msg string, args ...any) { log(ErrorLevel, msg, args...) }

// Log at [FatalLevel]
func Fatal(msg string, args ...any) { log(FatalLevel, msg, args...) }

// Log at given [slog.Level]
func Log(level slog.Level, msg string, args ...any) {
	log(level, msg, args...)
}

// copied and adapted from [log/slog] package
// log is the low-level logging method for methods that take ...any.
// It must always be called directly by an exported logging method
// or function, because it uses a fixed call depth to obtain the pc.
func log(lvl slog.Level, msg string, args ...any) {
	if lvl < minLevel {
		return
	}
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc := pcs[0]
	r := slog.NewRecord(time.Now(), lvl, msg, pc)
	r.Add(args...)
	ctx := context.Background()
	if termHandler.Enabled(ctx, lvl) {
		termHandler.Handle(ctx, r)
	}
	if fileHandler != nil && fileHandler.Enabled(ctx, lvl) {
		fileHandler.Handle(ctx, r)
	}
}
