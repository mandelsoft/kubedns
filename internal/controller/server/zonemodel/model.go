package zonemodel

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/mandelsoft/goutils/sliceutils"
	"github.com/mandelsoft/kubecrtutils/objutils"
	"github.com/mandelsoft/logging"
	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

type ZoneKey struct {
	reconcile.Request
	ClusterName string
}

func NewZoneKey(clusterName, namespace, name string) ZoneKey {
	return ZoneKey{
		ClusterName: clusterName,
		Request: reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}},
	}
}

func ZoneKeyFromObject(c string, obj client.Object) ZoneKey {
	return ZoneKey{ClusterName: c, Request: reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}}}
}

func ZoneRefKeyFromObject(c string, obj client.Object, ref string) ZoneKey {
	return ZoneKey{ClusterName: c, Request: reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      ref,
		Namespace: obj.GetNamespace(),
	}}}
}

func (k ZoneKey) String() string {
	return fmt.Sprint(k.ClusterName, "/", k.Request)
}

func (k ZoneKey) For(name string) ZoneKey {
	k.Name = name
	return k
}

type Source interface {
	GetName() string
}

type Index interface {
	LookupRelativeDomainName(src Source, zone ZoneKey, rel string) ([]corednsv1alpha1.CoreDNSEntry, error)
	LookupIP(src Source, zone ZoneKey, ip string) ([]corednsv1alpha1.CoreDNSEntry, error)
}

type Model struct {
	lock sync.RWMutex

	index Index
	zones map[ZoneKey]*Zone
}

func New(index Index) *Model {
	return &Model{index: index, zones: make(map[ZoneKey]*Zone)}
}

func (m *Model) GetZone(key ZoneKey) *Zone {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.zones[key]
}

func (m *Model) RemoveZone(zk ZoneKey) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.zones, zk)

}
func (m *Model) AddZone(s Source, key ZoneKey, zone *corednsv1alpha1.HostedZone) *Zone {
	m.lock.Lock()
	defer m.lock.Unlock()

	old := m.zones[key]
	if old == nil {
		old = NewZone(m, s, key)
		m.zones[key] = old

		for k, z := range m.zones {
			if z.parentName != "" {
				if key == k.For(z.parentName) {
					z.parent = old
					old.children[k.Name] = z
				}
			}
		}
	}
	id, err := strconv.ParseUint(zone.ObjectMeta.ResourceVersion, 10, 64)
	if err != nil {
		id = 653432456
	}
	old.serialId = uint32(id)

	if old.parent != nil {
		if old.parent.key.Name != zone.Spec.ParentRef {
			delete(old.parent.children, zone.Name)
		}
	}
	var parent *Zone
	if zone.Spec.ParentRef != "" {
		pkey := ZoneKey{
			Request:     reconcile.Request{objutils.RefObjectKeyFor(zone, zone.Spec.ParentRef)},
			ClusterName: s.GetName(),
		}
		parent = m.zones[pkey]
		if parent != nil {
			parent.children[pkey.Name] = old
		}

	}
	old.parent = parent
	old.parentName = zone.Spec.ParentRef
	if old.parentName == "" {
		old.names = sliceutils.Transform(zone.Spec.DomainNames, dns.Fqdn)
	} else {
		old.names = sliceutils.Transform(zone.Spec.DomainNames, Rdn)
	}

	for parent != nil {
		parent.serialId = old.serialId
		parent = parent.parent
	}
	return old
}

////////////////////////////////////////////////////////////////////////////////

type Zone struct {
	model      *Model
	serialId   uint32
	key        ZoneKey
	source     Source
	parent     *Zone
	parentName string
	children   map[string]*Zone
	names      []string
}

func NewZone(m *Model, s Source, key ZoneKey) *Zone {
	return &Zone{model: m, key: key, source: s, children: make(map[string]*Zone)}
}

func (z *Zone) GetSerialId() uint32 {
	return z.serialId
}

func (z *Zone) GetKey() ZoneKey {
	return z.key
}

func (z *Zone) GetSource() Source {
	return z.source
}

func (z *Zone) Matches(qname string) string {
	zone := ""
	for _, zname := range z.names {
		if dns.IsSubDomain(zname, qname) {
			// We want the *longest* matching zone, otherwise we may end up in a parent
			if len(zname) > len(zone) {
				zone = zname
			}
		}
	}
	return zone
}

type Info struct {
	Zone      *Zone
	ZoneNames []string

	Entries    []*corednsv1alpha1.CoreDNSEntry
	EntryNames []string
}

func (z *Zone) ResolveFQDN(logger logging.Logger, qname string) (*Info, error) {
	z.model.lock.RLock()
	defer z.model.lock.RUnlock()

	qname = dns.Fqdn(qname)
	zn := z.Matches(qname)
	if zn == "" {
		return nil, nil
	}

	sub := qname
	zone := z
	cur := ""
nextForward:
	for {
		sub = Rdn(sub[:len(sub)-len(zn)])
		cur = Join(zn, cur)
		logger.Info("lookup nested zone for {{current}}/{{relative}}", "current", cur, "relative", sub)
		for _, nested := range zone.children {
			r := nested.Matches(sub)
			if r != "" {
				zn = r
				zone = nested
				continue nextForward
			}
		}
		logger.Info("responsible zone {{zonekey}}: {{current}}/{{relative}}", "zonekey", zone.key, "current", cur, "relative", sub)
		break
	}

	var list []corednsv1alpha1.CoreDNSEntry
	var err error
	curz := cur
	labels := dns.SplitDomainName(sub)
	rel := ""
	for i := range labels {
		l := labels[len(labels)-1-i]
		cur = Join(l, cur)
		rel = Rdn(Join(l, rel))
		logger.Info("lookup NS record for {{current}}/{{relative}}", "current", cur, "relative", rel)
		list, err = zone.model.index.LookupRelativeDomainName(zone.source, zone.key, rel)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			if len(e.Spec.NS) != 0 {
				zn = cur
				logger.Info("found delegated zone {{zonekey}}: {{current}}/{{relative}}", "zonekey", client.ObjectKeyFromObject(&e), "current", cur, "relative", rel)
				return &Info{Zone: nil, ZoneNames: sliceutils.AsSlice(cur), Entries: []*corednsv1alpha1.CoreDNSEntry{&e}, EntryNames: sliceutils.AsSlice(cur)}, nil
				break
			}
		}

	}
	var entries []*corednsv1alpha1.CoreDNSEntry
	for _, e := range list {
		logger.Info("found entry {{entry}} for {{current}}[{{zonekey}}]/{{relative}}", "entry", client.ObjectKeyFromObject(&e), "zonekey", zone.key, "current", curz, "relative", sub)
		entries = append(entries, &e)
	}

	return &Info{Zone: zone, ZoneNames: sliceutils.AsSlice(curz), Entries: entries, EntryNames: sliceutils.AsSlice(cur)}, nil
}

func (z *Zone) ResolveIP(logger logging.Logger, ip string) ([]*Info, error) {
	z.model.lock.RLock()
	defer z.model.lock.RUnlock()

	return z.resolveIP([]string{""}, logger, ip)
}

func (z *Zone) resolveIP(fqdns []string, logger logging.Logger, ip string) ([]*Info, error) {

	list, err := z.model.index.LookupIP(z.source, z.key, ip)
	if err != nil {
		return nil, err
	}

	fqdns = expand(z.names, fqdns, nil)
	var result []*Info
	var entries []*corednsv1alpha1.CoreDNSEntry
	var names []string
	if len(list) > 0 {
		for _, e := range list {
			logger.Info("found entry {{entry}} for {{zonekey}}/{{ip}}", "entry", client.ObjectKeyFromObject(&e), "zonekey", z.key, "ip", ip)
			entries = append(entries, &e)
			names = expand(e.Spec.DNSNames, fqdns, names)
		}
		result = append(result, &Info{Zone: z, ZoneNames: fqdns, Entries: entries, EntryNames: names})
	}

	for _, s := range z.children {
		list, err := s.resolveIP(fqdns, logger, ip)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}

	return result, nil
}

func expand(names []string, fqnds []string, list []string) []string {
	for _, n := range names {
		for _, f := range fqnds {
			list = append(list, dns.Fqdn(n)+f)
		}
	}
	return list
}
