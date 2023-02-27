/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"github.com/loggie-io/loggie/pkg/core/cfg"
	"github.com/loggie-io/operator/pkg/config"
	"github.com/loggie-io/operator/pkg/webhook"
	runtimeWebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/loggie-io/loggie/pkg/core/log"
	logconfigv1beta1 "github.com/loggie-io/loggie/pkg/discovery/kubernetes/apis/loggie/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(logconfigv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var port int
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var certDir string
	var configPath string
	flag.IntVar(&port, "port", 9443, "Loggie Operator server port.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":9296", "The address the metric endpoint binds to.")
	flag.StringVar(&certDir, "cert-dir", "/tmp/cert", "cert-dir is the directory that contains the server key and certificate.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":9297", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configPath, "config-path", "config.yml", "Global Configuration path.")
	flag.Parse()

	log.InitDefaultLogger()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		CertDir:                certDir,
		Port:                   port,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5a8e7206.loggie.io",
	})
	if err != nil {
		log.Fatal("unable to start manager: %v", err)
	}

	// read configuration
	conf := config.Config{}
	unpack := cfg.UnPackFromFile(configPath, &conf)
	if err := unpack.Defaults().Validate().Do(); err != nil {
		log.Fatal("invalid config: %v, \n%s", err, unpack.Contents())
	}

	// TODO
	//if err = (&logconfig.Reconciler{
	//	Config: &conf,
	//	Client: mgr.GetClient(),
	//	Scheme: mgr.GetScheme(),
	//}).SetupWithManager(mgr); err != nil {
	//	log.Fatal("unable to create LogConfig controller: %v", err)
	//}

	if conf.Sidecar.Enabled {
		log.Info("sidecar injector is enabled")
		hookServer := mgr.GetWebhookServer()
		hookServer.Register("/mutate-inject-sidecar", &runtimeWebhook.Admission{Handler: &webhook.SidecarInjection{
			Client: mgr.GetClient(),
			Config: conf.Sidecar,
		}})
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatal("unable to set up health check: %v", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatal("unable to set up ready check: %v", err)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal("problem running manager: %v", err)
	}
}
