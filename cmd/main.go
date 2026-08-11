// Copyright 2026 DB-Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// DB Operator creates databases and make them available in the cluster via CRs.
package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/utils/thirdpartyapi"
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	kindarocksv1alpha1 "github.com/db-operator/db-operator/v2/api/v1alpha1"
	kindarocksv1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	webhookv1beta1 "github.com/db-operator/db-operator/v2/internal/webhook/v1beta1"

	"github.com/db-operator/db-operator/v2/internal/controller"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var CLI struct {
	Controller struct{} `cmd:"" help:"Start the db-operator controller"`
	Webhook    struct{} `cmd:"" help:"Start the db-operator webhook"`
	// Args
	// The address the metric endpoint binds to.
	MetricsBindAddress string `env:"DBO_METRICS_ADDR" default:":60000"`
	// The address the probe endpoint binds to.
	HealthProbeBindAddress string `env:"DBO_PROBE_ADDR" default:":8081"`
	// Enable leader election for controller manager.
	// Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool `env:"DBO_ENABLE_LEADER_ELECTION" default:"false"`
	// Set the logging level for db-operator.
	LogLevel string `env:"DBO_LOG_LEVEL" default:"info"`
	// If true, use development mode for Zap logger (more human-readable output).
	ZapDevel bool `env:"DBO_ZAP_DEVEL" default:"false"`
	// If true, db-operator will start with a profiler on port 54321.
	EnableProfiler bool `env:"DBO_ENABLE_PROFILER" default:"false"`
	// Path to the config file for db-operator.
	Config string `env:"DBO_CONFIG" default:"/srv/config/config.yaml"`
	// The interval at which the controller will reconcile the resources.
	ReconcileInterval time.Duration `env:"DBO_RECONCILE_INTERVAL" default:"30s"`
	// The namespaces that db-operator will watch for resources. If empty, all namespaces will be watched.
	WatchNamespaces []string `env:"DBO_WATCH_NAMESPACES"`
	// Enabling this will make the operator only reconcile when k8s objects were changed (currently used only by dbuser and database controllers).
	CheckForChanges bool `env:"DBO_CHECK_FOR_CHANGES" default:"false"`
}

var ErrUnknownCommand = errors.New("unknown command")

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kindarocksv1alpha1.AddToScheme(scheme))
	utilruntime.Must(kindarocksv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	thirdpartyapi.AppendToScheme(scheme)
}

func main() {
	kongCtx := kong.Parse(&CLI)

	// Prepare logger
	level, err := zapcore.ParseLevel(CLI.LogLevel)
	if err != nil {
		// Defaulting to the info level
		level = zapcore.InfoLevel
	}

	opts := zap.Options{
		Development: CLI.ZapDevel,
		Level:       level,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Enable profiler
	if CLI.EnableProfiler {
		setupLog.Info("Enabling profiler", "port", "54321")
		go func() {
			if err := http.ListenAndServe("localhost:54321", nil); err != nil {
				setupLog.Error(err, "Couldn't start profiler")
				os.Exit(1)
			}
		}()
	}

	// Configure webhook
	webhookSrv := webhook.NewServer(webhook.Options{
		Port: 9443,
	})

	// Configure manager
	setupLog.Info("Starting db-operator")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: CLI.MetricsBindAddress,
		},

		WebhookServer:          webhookSrv,
		HealthProbeBindAddress: CLI.HealthProbeBindAddress,
		LeaderElection:         CLI.EnableLeaderElection,
		LeaderElectionID:       "6fe36c14.kinda.rocks",
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	switch kongCtx.Command() {
	case "controller":
		setupLog.Info("Starting controller")

		conf, err := config.LoadConfig(CLI.Config)
		if err != nil {
			setupLog.Error(err, "An error occurred when reading the config")
			os.Exit(1)
		}

		setupLog.Info("Registering DbInstance controller")
		if err = (&controller.DbInstanceReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("DbInstance"),
			Scheme:   mgr.GetScheme(),
			Interval: time.Duration(CLI.ReconcileInterval),
			Recorder: mgr.GetEventRecorder("dbinstance-controller"),
			Conf:     conf,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "DbInstance")
			os.Exit(1)
		}

		if len(CLI.WatchNamespaces) > 0 {
			setupLog.Info("Database resources will be served in the next namespaces", "namespaces", CLI.WatchNamespaces)
		}

		setupLog.Info("Registering Database controller")

		if err = (&controller.DatabaseReconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("Database"),
			Scheme:          mgr.GetScheme(),
			Recorder:        mgr.GetEventRecorder("database-controller"),
			Interval:        time.Duration(CLI.ReconcileInterval),
			Conf:            conf,
			WatchNamespaces: CLI.WatchNamespaces,
			CheckChanges:    CLI.CheckForChanges,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "Database")
			os.Exit(1)
		}

		setupLog.Info("Registering DbUser controller")
		if err = (&controller.DbUserReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorder("dbuser-controller"),
			Interval:     time.Duration(CLI.ReconcileInterval),
			CheckChanges: CLI.CheckForChanges,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "DbUser")
			os.Exit(1)
		}
	case "webhook":
		setupLog.Info("Starting webhook server")
		setupLog.Info("Registering Database webhook")
		// nolint:goconst
		if err := webhookv1beta1.SetupDatabaseWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create webhook", "webhook", "database")
			os.Exit(1)
		}
		setupLog.Info("Registering DbInstance webhook")
		// nolint:goconst
		if err := webhookv1beta1.SetupDbInstanceWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create webhook", "webhook", "DbInstance")
			os.Exit(1)
		}
		setupLog.Info("Registering DbUser webhook")
		// nolint:goconst
		if err := webhookv1beta1.SetupDbUserWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create webhook", "webhook", "DbUser")
			os.Exit(1)
		}

	default:
		setupLog.Error(ErrUnknownCommand, "Unknown command is provided")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	setupLog.Info("Registering probe endpoints")
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}
