package internal

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log-collector-agent/internal/utils"
	"net"
	"net/http"
	"strconv"
	"time"
)

type LogCollectorAgentServer struct {
}

type Options struct {
	Port int
}

func (s LogCollectorAgentServer) RUN(options Options) {
	stopCh := make(chan struct{})
	go s.serveHealthProbes(options.Port)
	kubeClient, _ := buildKubeClient()

	ls := LogService{}
	ls.RUN(func() {
		listOptions := informers.WithTweakListOptions(func(options *v1.ListOptions) {
		})
		informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 10*time.Minute, listOptions)
		logStructController, _ := NewServerLogStructController(options, stopCh, informerFactory, *kubeClient)
		informerFactory.Start(stopCh)
		go logStructController.RUN(context.TODO(), 1)
	})

	<-stopCh
}

func buildKubeClient() (*kubernetes.Clientset, error) {
	kubeConfigDir := ""
	if utils.IsDebugEnv() {
		kubeConfigDir = homedir.HomeDir() + "/.kube/config"
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigDir)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (s LogCollectorAgentServer) serveHealthProbes(port int) error {
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Handler:           mux,
		MaxHeaderBytes:    1 << 20,
		IdleTimeout:       90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadHeaderTimeout: 32 * time.Second,
	}

	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
		time.Sleep(100 * time.Second)
	})

	listen, _ := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(port)))
	err := httpServer.Serve(listen)
	if err != nil {
		return err
	}
	return nil
}
