package replicate

import (
	"github.com/mandelsoft/kubedns/internal/controller/common"
)

const SOURCE = "source"
const TARGET = "target"

const GROUP = "replication"

const ControllerEntry = "replication.corednsentry"
const ControllerHostedzone = "replication.hostedzone"

const REPLICATED_ANNOTATION = "coredns.mandelsoft.org/replication"

const IndexKeyZoneParent = common.IndexKeyZoneParent
const IndexKeyEntryZone = common.IndexKeyEntryZone
