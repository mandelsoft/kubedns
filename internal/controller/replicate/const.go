package replicate

const SOURCE = "source"
const TARGET = "target"

const GROUP = "replication"
const ENTRY_GROUP = "entry-replication"
const HOSTEDZONE_GROUP = "hostedzone-replication"

const ControllerEntry = "replication.corednsentry"
const ControllerHostedzone = "replication.hostedzone"

const REPLICATED_ANNOTATION = "coredns.mandelsoft.org/replication"

const IndexKeyZoneParent = "replication.corednsentry.zone"
const IndexKeyEntryZone = "replication.corednsentry.entries"
