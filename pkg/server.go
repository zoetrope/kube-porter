package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	socketAddr  string
	kubeconfig  string
	manifest    string
	logFilePath string
	logger      *zap.Logger

	reconciler *manifestReconciler
}

func NewServer(socketAddr string, kubeconfig string, manifest string, logFilePath string) *Server {
	reconciler := newManifestReconciler(kubeconfig, manifest)
	return &Server{
		socketAddr:  socketAddr,
		kubeconfig:  kubeconfig,
		manifest:    manifest,
		logFilePath: logFilePath,
		logger:      zap.L().Named("server"),
		reconciler:  reconciler,
	}
}

func (s Server) Run() error {
	listener, err := net.Listen("unix", s.socketAddr)
	if err != nil {
		s.logger.Error("failed to listen", zap.Error(err))
		return err
	}

	mux := http.NewServeMux()
	hs := &http.Server{
		Handler: mux,
	}

	ctx, cancel := context.WithCancel(context.Background())

	mux.HandleFunc("/", handle)
	mux.HandleFunc("/ready", ready)
	mux.HandleFunc("/status", s.getForwarderList)
	mux.HandleFunc("/logfile", s.getLogFilePath)
	mux.HandleFunc("/stop", func(_ http.ResponseWriter, _ *http.Request) {
		cancel()
	})

	go func() {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigCh:
		case <-ctx.Done():
		}
		defer cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		err = s.reconciler.run(ctx)
		if err != nil {
			s.logger.Error("failed to reconcile", zap.Error(err))
		}
		s.logger.Info("reconcile is done")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = hs.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("failed to serve", zap.Error(err))
		}
		s.logger.Info("server shutdown")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := hs.Shutdown(shutdownContext); err != nil {
			s.logger.Error("failed to shutdown", zap.Error(err))
		}
		s.logger.Info("shutdown is done")
	}()

	wg.Wait()

	fmt.Println("done")
	return nil
}

func (s Server) renderJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.logger.Error("failed to output JSON", zap.Error(err))
	}
}
func handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "hello")
}

func ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ok")
}

type ForwarderStatus struct {
	Target     `json:,inline`
	Forwarding bool `json:"forwarding"`
}

func (s Server) getForwarderList(w http.ResponseWriter, r *http.Request) {
	s.renderJSON(w, s.reconciler.Status(), http.StatusOK)
}

func (s Server) getLogFilePath(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, s.logFilePath)
}
