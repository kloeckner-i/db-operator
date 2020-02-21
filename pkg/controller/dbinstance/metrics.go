package dbinstance

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promDBInstancesPhase = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "db_operator",
		Subsystem: "dbinstance",
		Name:      "phase",
		Help:      "Return information about the phase of a database instances (cr) object",
	},
		[]string{
			"dbinstance",
		})

	promDBInstancesPhaseTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "db_operator",
		Subsystem:  "handler",
		Name:       "dbinstance_seconds",
		Help:       "Return Summary over the internal timing of the phase of the database instance object",
		Objectives: map[float64]float64{},
	},
		[]string{
			"phase",
		})
)

func dbInstancePhaseToFloat64(phase string) float64 {
	phaseMap := map[string]float64{
		"default":        -10,
		"":               0,
		phaseValidate:    10,
		phaseCreate:      20,
		phaseBroadcast:   -25,
		phaseProxyCreate: 50,
		phaseRunning:     100,
	}

	if _, found := phaseMap[phase]; found {
		return phaseMap[phase]
	}

	return phaseMap["default"]
}
