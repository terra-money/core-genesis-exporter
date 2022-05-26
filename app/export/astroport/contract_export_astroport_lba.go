package astroport

import (
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

// no need to count LBA, it'll be covered by ASTRO-UST LP.

func ExportAstroportLBA(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	return nil, nil
}
