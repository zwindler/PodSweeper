// Package main is the entry point for the PodSweeper Gamemaster controller.
// The Gamemaster is responsible for:
// - Managing the game grid (spawning/deleting pods)
// - Tracking game state (mines, revealed cells, level)
// - Handling game logic (BFS propagation, victory/defeat detection)
// - Watching ConfigMap for game configuration
// - Running the admission webhook for advanced levels
package main

import (
	"context"
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/zwindler/podsweeper/internal/controller"
	"github.com/zwindler/podsweeper/pkg/game"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// getEnvOrDefault returns the value of the environment variable or a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var namespace string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&namespace, "namespace", game.DefaultNamespace, "The namespace to watch for game pods.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "podsweeper-gamemaster",
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	// Create game state store (persisted in Kubernetes Secret)
	store := game.NewSecretStore(mgr.GetClient(),
		game.WithNamespace(namespace),
	)

	// Configure container images (can be overridden via environment variables)
	imageConfig := controller.ImageConfig{
		HintAgent: getEnvOrDefault("PODSWEEPER_HINT_IMAGE", controller.DefaultHintAgentImage),
		Explosion: getEnvOrDefault("PODSWEEPER_EXPLOSION_IMAGE", controller.DefaultExplosionImage),
		Victory:   getEnvOrDefault("PODSWEEPER_VICTORY_IMAGE", controller.DefaultVictoryImage),
	}

	// Create and register the game controller
	gameController := controller.NewGameController(mgr.GetClient(), controller.GameControllerConfig{
		Namespace: namespace,
		Store:     store,
		Images:    imageConfig,
	})

	if err := gameController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GameController")
		os.Exit(1)
	}

	// Create and register the config controller (watches ConfigMap for game configuration)
	configController := controller.NewConfigController(mgr.GetClient(), controller.ConfigControllerConfig{
		Namespace: namespace,
		Store:     store,
	})

	if err := configController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigController")
		os.Exit(1)
	}

	// Ensure the config ConfigMap exists
	ctx := context.Background()
	if err := configController.EnsureConfigMap(ctx); err != nil {
		setupLog.Error(err, "failed to ensure config ConfigMap exists")
		// Don't exit - the ConfigMap might be created later
	}

	// TODO: Set up admission webhook (for levels 5+)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting gamemaster",
		"namespace", namespace,
		"probeAddr", probeAddr,
		"hintImage", imageConfig.HintAgent,
		"explosionImage", imageConfig.Explosion,
		"victoryImage", imageConfig.Victory,
	)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
