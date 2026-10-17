package configcommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/configcatalog"
)

// DescribeConfig handles config/describe: the catalog of everything that can be configured.
func DescribeConfig() rpcserver.Method {
	return func(context.Context, rpcserver.Call) (any, error) {
		return configcatalog.Describe(), nil
	}
}
