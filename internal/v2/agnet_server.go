package v2

import (
	v1 "github.com/yshaojie/log-collector/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	manager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type LogCollectorAgentServer struct {
}

var (
	scheme = runtime.NewScheme()
	logger = log.Log.WithName("log-agent")
)

func init() {
	log.SetLogger(zap.New())
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1.AddToScheme(scheme))
}

func (s *LogCollectorAgentServer) RUN() {

	logger.Info("setting up manager...")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme:           scheme,
		LeaderElection:   false,
		LeaderElectionID: "a661852b.4yxy.io",
	})
	if err != nil {
		logger.Error(err, " set manager fail. ")
		os.Exit(1)
	}
	needLeaderElection := false
	ctl, err := controller.New("log-agent", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		CacheSyncTimeout:        0,
		RecoverPanic:            nil,
		NeedLeaderElection:      &needLeaderElection,
		Reconciler:              NewServerlogController(mgr.GetClient()),
		RateLimiter:             nil,
		LogConstructor:          nil,
	})
	if err != nil {
		logger.Error(err, " create controller fail. ")
		os.Exit(1)
	}

	err = ctl.Watch(source.Kind(mgr.GetCache(), &v1.ServerLog{}),
		&handler.EnqueueRequestForObject{},
		predicate.NewPredicateFuncs(func(object client.Object) bool {
			//object.GetManagedFields()
			serverLog := object.(*v1.ServerLog)
			nodeName := os.Getenv("NODE_NAME")
			//true:需要处理
			//只处理跟本节点匹配的ServerLog
			return len(nodeName) > 0 && serverLog.Spec.NodeName == nodeName
		}))
	if err != nil {
		logger.Error(err, " create Watch fail. ")
		os.Exit(1)
	}
	logger.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
