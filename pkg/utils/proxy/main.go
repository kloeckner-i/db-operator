//+build !tests

package proxy

import "github.com/kloeckner-i/db-operator/pkg/config"

func init() {
	conf = config.LoadConfig()
}
