package database

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promDBsPhase = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "db_operator",
		Subsystem: "database",
		Name:      "phase",
		Help:      "Return information about the phase of a database (cr) object",
	},
		[]string{
			"db_namespace",
			"dbinstance",
			"database",
		})

	promDBsStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "db_operator",
		Subsystem: "database",
		Name:      "status",
		Help:      "Return information about the status of a database (cr) object",
	},
		[]string{
			"db_namespace",
			"dbinstance",
			"database",
		})

	promDBsPhaseTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "db_operator",
		Subsystem:  "handler",
		Name:       "database_seconds",
		Help:       "Return Summary over the internal timing of the phase functions of the database object",
		Objectives: map[float64]float64{},
	},
		[]string{
			"phase",
		})

	promDBsPhaseError = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "db_operator",
		Subsystem: "handler",
		Name:      "database_phase_error",
		Help:      "Count errors in reconcile cycle",
	},
		[]string{
			"phase",
		})
)

func dbPhaseToFloat64(phase string) float64 {
	phaseMap := map[string]float64{
		"default":                 -10,
		"":                        0,
		phaseCreate:               10,
		phaseConfigMap:            20,
		phaseInstanceAccessSecret: 25,
		phaseProxy:                30,
		phaseBackupJob:            40,
		phaseMonitoring:           45,
		phaseFinish:               50,
		phaseReady:                100,
	}

	if _, found := phaseMap[phase]; found {
		return phaseMap[phase]
	}

	return phaseMap["default"]
}

func boolToFloat64(b bool) float64 {
	if b {
		return float64(1)
	}
	return float64(0)
}
