//+build !tests

package backup

import "github.com/kloeckner-i/db-operator/pkg/config"

func init() {
	conf = config.LoadConfig()
}
