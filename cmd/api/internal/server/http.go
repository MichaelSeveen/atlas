package server

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
}

func DefaultHTTPConfig(address string) HTTPConfig {
	return HTTPConfig{
		Address:           address,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
}

func NewHTTPServer(handler http.Handler, config HTTPConfig) (*http.Server, error) {
	if handler == nil || config.Address == "" {
		return nil, errors.New("HTTP handler and address are required")
	}
	if config.ReadHeaderTimeout <= 0 || config.ReadTimeout <= 0 ||
		config.WriteTimeout <= 0 || config.IdleTimeout <= 0 || config.ShutdownTimeout <= 0 {
		return nil, errors.New("all HTTP deadlines must be positive")
	}
	if config.MaxHeaderBytes < 1024 || config.MaxHeaderBytes > 1<<20 {
		return nil, errors.New("HTTP header limit is outside the foundation policy")
	}

	return &http.Server{
		Addr:              config.Address,
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
		// net/http's legacy error logger can include raw connection details and
		// is not compatible with the closed foundation schema. Terminal server
		// failures return from ListenAndServe and use the structured failure path.
		ErrorLog: log.New(io.Discard, "", 0),
	}, nil
}
