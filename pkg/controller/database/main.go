//+build !tests

package database

import (
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func init() {
	metrics.Registry.MustRegister(promDBsPhaseTime, promDBsPhase, promDBsStatus, promDBsPhaseError)
}
