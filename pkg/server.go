package pkg

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Server struct {
	socketAddr string
	kubeconfig string
	config     string
	logger     *zap.Logger
	forwarders map[string]*Forwarder
}

func NewServer(socketAddr string, kubeconfig string, config string) *Server {
	return &Server{
		socketAddr: socketAddr,
		kubeconfig: kubeconfig,
		config:     config,
		logger:     zap.L().Named("server"),

		forwarders: make(map[string]*Forwarder),
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

		err = s.reconcile(ctx)
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

func (s Server) reconcile(ctx context.Context) error {
	err := s.forward(ctx)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(s.config); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcher.Events:
			err = s.forward(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (s Server) forward(ctx context.Context) error {
	kcfg, err := clientcmd.BuildConfigFromFlags("", s.kubeconfig)
	if err != nil {
		return err
	}

	kcfg.APIPath = "/api"
	kcfg.GroupVersion = &corev1.SchemeGroupVersion
	kcfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	restClient, err := rest.RESTClientFor(kcfg)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig(s.config)
	if err != nil {
		return err
	}
	for _, target := range cfg.Targets {
		if _, ok := s.forwarders[target.String()]; ok {
			//TODO
			continue
		}
		f := NewForwarder(kcfg, restClient, target)
		go func() {
			//TODO
			err = f.Run(ctx)
			if err != nil {
				s.logger.Error("failed to run", zap.Error(err))
			}
		}()
		s.forwarders[target.String()] = f
	}
	return nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "hello")
}

func ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ok")
}