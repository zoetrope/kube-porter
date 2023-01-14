package pkg

import (
	"context"
	"sort"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type manifestReconciler struct {
	kubeconfig string
	manifest   string
	logger     *zap.Logger

	mu         sync.RWMutex
	forwarders map[string]*Forwarder
}

func newManifestReconciler(kubeconfig string, manifest string) *manifestReconciler {
	return &manifestReconciler{
		kubeconfig: kubeconfig,
		manifest:   manifest,
		logger:     zap.L().Named("manifest-reconciler"),

		mu:         sync.RWMutex{},
		forwarders: make(map[string]*Forwarder),
	}
}

func (r manifestReconciler) run(ctx context.Context) error {
	err := r.reconcile(ctx)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(r.manifest); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcher.Events:
			err = r.reconcile(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (r manifestReconciler) reconcile(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, err := LoadManifest(r.manifest)
	if err != nil {
		return err
	}

OUTER:
	for k, f := range r.forwarders {
		for _, target := range cfg.Targets {
			if k == target.String() {
				continue OUTER
			}
		}
		f.Stop()
		delete(r.forwarders, k)
	}

	for _, target := range cfg.Targets {
		if _, ok := r.forwarders[target.String()]; ok {
			continue
		}
		f, err := NewForwarder(r.kubeconfig, target)
		if err != nil {
			return err
		}
		f.Run(ctx)
		r.forwarders[target.String()] = f
	}
	return nil
}

func (r manifestReconciler) Status() []ForwarderStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var forwarderList []ForwarderStatus
	for _, forwarder := range r.forwarders {
		forwarderList = append(forwarderList, ForwarderStatus{
			Target:     forwarder.target,
			Forwarding: forwarder.isForwarding(),
		})
	}
	sort.Slice(forwarderList, func(i, j int) bool {
		return forwarderList[i].String() < forwarderList[j].String()
	})
	return forwarderList
}
