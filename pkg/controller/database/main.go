//+build !tests

package database

import (
	"github.com/kloeckner-i/db-operator/pkg/config"

	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func init() {
	conf = config.LoadConfig()
	metrics.Registry.MustRegister(promDBsPhaseTime, promDBsPhase, promDBsStatus, promDBsPhaseError)
}
