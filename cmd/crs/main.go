package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/saltyorg/crs/build"
	"github.com/saltyorg/crs/web"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type config struct {
	Web web.Config `yaml:"web"`
}

var (
	// CLI
	cli struct {
		globals

		// flags
		Config    string `type:"path" default:"${config_file}" short:"c" env:"APP_CONFIG" help:"Config file path"`
		Upload    string `type:"path" default:"${upload_path}" short:"u" env:"APP_UPLOAD" help:"Upload file path"`
		Log       string `type:"path" default:"${log_file}" short:"l" env:"APP_LOG" help:"Log file path"`
		Verbosity int    `type:"counter" default:"0" short:"v" env:"APP_VERBOSITY" help:"Log level verbosity"`

		Host string `type:"string" default:"0.0.0.0" short:"h" env:"APP_HOST" help:"Host to listen on"`
		Port int    `type:"number" default:"8684" short:"p" env:"APP_PORT" help:"Port to listen on"`
	}
)

type globals struct {
	Version versionFlag `name:"version" help:"Print version information and quit"`
}

func main() {
	// cli
	ctx := kong.Parse(&cli,
		kong.Name("crs"),
		kong.Description("Cloudbox Restore Service"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Summary: true,
			Compact: true,
		}),
		kong.Vars{
			"version":     fmt.Sprintf("%s (%s@%s)", build.Version, build.GitCommit, build.Timestamp),
			"config_file": filepath.Join(defaultConfigDirectory("crs", "config.yml"), "config.yml"),
			"upload_path": filepath.Join(defaultConfigDirectory("crs", "config.yml"), "uploads"),
			"log_file":    filepath.Join(defaultConfigDirectory("crs", "config.yml"), "activity.log"),
		},
	)

	if err := ctx.Validate(); err != nil {
		fmt.Println("Failed parsing cli:", err)
		return
	}

	// logger
	logger := log.Output(io.MultiWriter(zerolog.ConsoleWriter{
		TimeFormat: time.Stamp,
		Out:        os.Stderr,
		NoColor:    runtime.GOOS == "windows",
	}, zerolog.ConsoleWriter{
		TimeFormat: time.Stamp,
		Out: &lumberjack.Logger{
			Filename:   cli.Log,
			MaxSize:    5,
			MaxAge:     14,
			MaxBackups: 5,
		},
		NoColor: true,
	}))

	switch {
	case cli.Verbosity == 1:
		log.Logger = logger.Level(zerolog.DebugLevel)
	case cli.Verbosity > 1:
		log.Logger = logger.Level(zerolog.TraceLevel)
	default:
		log.Logger = logger.Level(zerolog.InfoLevel)
	}

	// config
	log.Trace().Msg("Initialising config")
	file, err := os.Open(cli.Config)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed opening config")
		return
	}
	defer file.Close()

	cfg := config{}
	decoder := yaml.NewDecoder(file, yaml.Strict())
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Error().Msg("Failed decoding configuration")
		log.Error().Msg(err.Error())
		return
	}

	// upload
	if _, err := os.Stat(cli.Upload); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed validating upload directory")
	}

	// display initialised banner
	log.Info().
		Str("version", fmt.Sprintf("%s (%s@%s)", build.Version, build.GitCommit, build.Timestamp)).
		Msg("Initialised")

	// web server
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	wc := web.New(&cfg.Web, cli.Upload)

	r.Use(gin.Recovery())
	r.Use(cors.Default())
	r.Use(wc.Logger())

	wc.SetHandlers(r)

	// run web server
	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", cli.Host, cli.Port),
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().
				Err(err).
				Msg("Failed starting web server")
		}
	}()

	log.Info().
		Str("host", cli.Host).
		Int("port", cli.Port).
		Msg("Listening for requests")

	// wait for shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn().Msg("Shutting down...")
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(sctx); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed graceful webserver shutdown")
	}

	select {
	case <-sctx.Done():
		break
	}
}
