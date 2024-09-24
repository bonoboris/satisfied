package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/bonoboris/satisfied/app"
	"github.com/bonoboris/satisfied/log"
)

// Data are embedded in the executable

//go:embed assets
var assets embed.FS

var (
	fs         *flag.FlagSet
	verbose    *bool
	vverbose   *bool
	quiet      *bool
	fps        *int
	cpuprofile *string
	memprofile *string
)

const usage = `Usage: %s [options] [FILE]

FILE is an optional path to a satisfied project file to load.

Options:
`

func init() {
	fs = flag.NewFlagSet("satisfied", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	quiet = fs.Bool("q", false, "WARN verbosity")
	verbose = fs.Bool("v", false, "DEBUG verbosity")
	vverbose = fs.Bool("vv", false, "TRACE verbosity")
	fps = fs.Int("fps", app.DefaultTargetFPS, "Target / Max FPS, (use a low value when using -vv to reduce the ammount of logs)")

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	fs.Usage = func() {
		binName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, usage, binName)
		fs.PrintDefaults()
	}
}

func parseArgs() (slog.Level, *app.AppOptions) {
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	opts := &app.AppOptions{}
	logLevel := log.InfoLevel
	switch {
	case *vverbose:
		logLevel = log.TraceLevel
	case *verbose:
		logLevel = log.DebugLevel
	case *quiet:
		logLevel = log.WarnLevel
	default:
		logLevel = log.InfoLevel
	}
	if fs.NArg() > 0 {
		opts.File = app.NormalizePath(fs.Arg(0))
	}
	opts.Fps = *fps
	return logLevel, opts
}

func main() {
	logLevel, opts := parseArgs()
	log.Init(logLevel, true)

	app.Init(assets, opts)

	if (cpuprofile != nil && *cpuprofile != "") || (memprofile != nil && *memprofile != "") {
		go func() {
			log.Info("starting http server", "addr", "localhost:6060")
			http.ListenAndServe("localhost:6060", nil)
		}()
	}

	if cpuprofile != nil && *cpuprofile != "" {
		if *cpuprofile == "" {
			*cpuprofile = "cpu.prof"
		}
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Error("cannot create cpu profile", "path", *cpuprofile, "err", err)
			os.Exit(1)
		}
		defer f.Close()
		log.Info("writing cpu profile", "path", *cpuprofile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for !app.ShouldExit() {
		app.Step()
	}
	app.Close()

	if memprofile != nil && *memprofile != "" {
		if *memprofile == "" {
			*memprofile = "mem.prof"
		}
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Error("cannot create mem profile", "path", *memprofile, "err", err)
			os.Exit(1)
		}
		defer f.Close()
		log.Info("writing mem profile", "path", *memprofile)
		pprof.WriteHeapProfile(f)
	}
}
