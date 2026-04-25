package direct

import (
	"github.com/mandelsoft/kubedns/internal/controller/common"
)

const ControllerHostedzone = "hostedzone"
const IndexKeyZoneParent = common.IndexKeyZoneParent

const ControllerEntry = "corednsentry"
const IndexKeyEntryZone = common.IndexKeyEntryZone

const GROUP = "operator"

var ParentIndexer = common.ParentIndexer
var ZoneIndexer = common.ZoneIndexer
