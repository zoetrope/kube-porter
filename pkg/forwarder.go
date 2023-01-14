package pkg

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/kubectl/pkg/util/podutils"
)

type Forwarder struct {
	config     *rest.Config
	restClient rest.Interface
	target     Target
	logger     *zap.Logger
	forwarding atomic.Bool
	cancel     context.CancelFunc
	exitCh     chan bool
}

func NewForwarder(kubeconfig string, target Target) (*Forwarder, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Forwarder{
		config:     config,
		restClient: restClient,
		target:     target,
		logger:     zap.L().Named(target.Name),
	}, nil
}

func (f *Forwarder) Run(ctx context.Context) {
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		f.cancel = cancel

		timeout := 1 * time.Second
		for {
			err := f.forward(ctx)
			if err != nil {
				if timeout < 30*time.Second {
					timeout *= 2
				}
				f.logger.Error("failed to forward", zap.Error(err))
			} else {
				timeout = 1 * time.Second
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):
			}
		}
	}()
}

func (f *Forwarder) Stop() {
	select {
	case <-f.exitCh:
		return
	default:
		close(f.exitCh)
		f.cancel()
	}
}

func (f *Forwarder) isStopped() bool {
	select {
	case <-f.exitCh:
		return true
	default:
		return false
	}
}

func (f *Forwarder) getObject(ctx context.Context, clientset *kubernetes.Clientset) (runtime.Object, error) {

	var obj runtime.Object
	var err error

	switch f.target.ObjectType {
	case "Deployment":
		obj, err = clientset.AppsV1().Deployments(f.target.Namespace).Get(ctx, f.target.Name, metav1.GetOptions{})
	case "StatefulSet":
		obj, err = clientset.AppsV1().StatefulSets(f.target.Namespace).Get(ctx, f.target.Name, metav1.GetOptions{})
	case "Service":
		obj, err = clientset.CoreV1().Services(f.target.Namespace).Get(ctx, f.target.Name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (f *Forwarder) forward(ctx context.Context) error {
	defer f.forwarding.Store(false)
	clientset, err := kubernetes.NewForConfig(f.config)
	if err != nil {
		return err
	}

	obj, err := f.getObject(ctx, clientset)
	if err != nil {
		return err
	}

	namespace, selector, err := polymorphichelpers.SelectorsForObject(obj)
	if err != nil {
		f.logger.Error("cannot attach to", zap.String("type", fmt.Sprintf("%T", obj)), zap.Error(err))
		return err
	}
	sortBy := func(pods []*corev1.Pod) sort.Interface { return sort.Reverse(podutils.ActivePods(pods)) }
	pod, _, err := polymorphichelpers.GetFirstPod(clientset.CoreV1(), namespace, selector.String(), 1*time.Second, sortBy)
	if err != nil {
		f.logger.Error("failed to get first pod", zap.Error(err))
		return err
	}
	//TODO: check pod status and rbac

	var ports []string
	if f.target.ObjectType == "Service" {
		ports, err = translatePorts(f.target.Ports, obj.(*corev1.Service), pod)
		f.logger.Info("translated ports", zap.Strings("orig", f.target.Ports), zap.Strings("translated", ports))
	} else {
		ports = f.target.Ports
	}
	f.logger.Info("found pod", zap.String("pod", pod.Namespace+"/"+pod.Name))

	req := f.restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(f.config)
	if err != nil {
		f.logger.Error("failed to RoundTripperFor", zap.Error(err))
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	l := f.logger.WithOptions(zap.WithCaller(false)).With(zap.String("forwarder", f.target.String()))
	stdoutLogger := &zapio.Writer{Log: l, Level: zap.InfoLevel}
	stderrLogger := &zapio.Writer{Log: l, Level: zap.ErrorLevel}
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	fw, err := portforward.NewOnAddresses(dialer, []string{"localhost"}, ports, stopChan, readyChan, stdoutLogger, stderrLogger)
	if err != nil {
		f.logger.Error("failed to NewOnAddresses", zap.Error(err))
		return err
	}
	go func() {
		//TODO: exit goroutine when connection lost
		<-ctx.Done()
		f.logger.Info("stop forwarding")
		stopChan <- struct{}{}
	}()

	f.forwarding.Store(true)
	f.logger.Info("start forwarding")
	err = fw.ForwardPorts()
	if err != nil {
		f.logger.Error("failed to ForwardPorts", zap.Error(err))
		return err
	}
	f.logger.Info("lost connection")
	return nil
}

func (f *Forwarder) isForwarding() bool {
	return f.forwarding.Load()
}

func translatePorts(ports []string, svc *corev1.Service, pod *corev1.Pod) ([]string, error) {
	var translated []string
	for _, port := range ports {
		var localPort, remotePort string

		parts := strings.Split(port, ":")
		if len(parts) == 1 {
			localPort = parts[0]
			remotePort = parts[0]
		} else if len(parts) == 2 {
			localPort = parts[0]
			remotePort = parts[1]
		} else {
			return nil, fmt.Errorf("invalid port: %s", port)
		}

		portnum, err := strconv.Atoi(remotePort)
		if err != nil {
			svcPort, err := util.LookupServicePortNumberByName(*svc, remotePort)
			if err != nil {
				return nil, err
			}
			portnum = int(svcPort)

			if localPort == remotePort {
				localPort = strconv.Itoa(portnum)
			}
		}
		containerPort, err := util.LookupContainerPortNumberByServicePort(*svc, *pod, int32(portnum))
		if err != nil {
			return nil, err
		}

		remotePort = strconv.Itoa(int(containerPort))
		if localPort != remotePort {
			translated = append(translated, fmt.Sprintf("%s:%s", localPort, remotePort))
		} else {
			translated = append(translated, remotePort)
		}
	}
	return translated, nil
}
