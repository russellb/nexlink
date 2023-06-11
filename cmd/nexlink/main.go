package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func addService(obj interface{}) {
	_ = obj.(*corev1.Service)
}

func updateService(oldObj interface{}, newObj interface{}) {
	_ = oldObj.(*corev1.Service)
	_ = newObj.(*corev1.Service)
}

func deleteService(obj interface{}) {
	_ = obj.(*corev1.Service)
}

func nexlink(c *cli.Context) error {
	var logger *zap.Logger
	var err error
	if c.Bool("debug") {
		logCfg := zap.NewDevelopmentConfig()
		logger, err = logCfg.Build()
		logger.Info("Debug logging enabled")
	} else {
		logCfg := zap.NewProductionConfig()
		logCfg.DisableStacktrace = true
		logger, err = logCfg.Build()
	}
	if err != nil {
		logger.Fatal(err.Error())
	}

	config, err := clientcmd.BuildConfigFromFlags("", c.String("kubeconfig"))
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err.Error())
	}

	// stop signal for the informer
	stopper := make(chan struct{})
	defer close(stopper)

	factory := informers.NewSharedInformerFactory(clientset, 0)
	svcInformer := factory.Core().V1().Services()
	informer := svcInformer.Informer()

	defer runtime.HandleCrash()

	// start informer ->
	go factory.Start(stopper)

	// start to sync and call list
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		err = fmt.Errorf("Timed out waiting for caches to sync")
		runtime.HandleError(err)
		return err
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addService,
		UpdateFunc: updateService,
		DeleteFunc: deleteService,
	})

	lister := svcInformer.Lister().Services(c.String("namespace"))

	svcs, err := lister.List(labels.Everything())

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("svcs:", svcs)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	select {
	case <-ctx.Done():
		break
	case <-stopper:
		break
	}
	return nil
}

func main() {
	var home string
	if home = homedir.HomeDir(); home == "" {
		home = os.Getenv("HOME")
	}
	app := &cli.App{
		Name:                 "nexlink",
		Usage:                "nexlink is a tool for linking services between kubernetes clusters",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "kubeconfig",
				Value:    filepath.Join(home, ".kube", "config"),
				Usage:    "absolute path to the kubeconfig file",
				EnvVars:  []string{"KUBECONFIG"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "namespace",
				Value:    "default",
				Usage:    "namespace to watch",
				EnvVars:  []string{"NAMESPACE"},
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "debug",
				Value:    false,
				Usage:    "enable debug logging",
				EnvVars:  []string{"NEXLINK_DEBUG"},
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			return nexlink(c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err.Error())
	}
}
