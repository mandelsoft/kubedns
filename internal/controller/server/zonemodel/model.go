package zonemodel

import (
	"context"
	"fmt"
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
	LookupRelativeDomainName(ctx context.Context, zone ZoneKey, rel string) ([]corednsv1alpha1.CoreDNSEntry, error)
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
	return old
}

////////////////////////////////////////////////////////////////////////////////

type Zone struct {
	model      *Model
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
	Zone     *Zone
	ZoneName string

	Entries   []*corednsv1alpha1.CoreDNSEntry
	EntryName string
}

func (z *Zone) Resolve(logger logging.Logger, qname string) (*Info, error) {
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
		list, err = zone.model.index.LookupRelativeDomainName(nil, zone.key, rel)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			if len(e.Spec.NS) != 0 {
				zn = cur
				logger.Info("found delegated zone {{zonekey}}: {{current}}/{{relative}}", "zonekey", client.ObjectKeyFromObject(&e), "current", cur, "relative", rel)
				return &Info{Zone: zone, ZoneName: curz, Entries: []*corednsv1alpha1.CoreDNSEntry{&e}, EntryName: cur}, nil
				break
			}
		}

	}
	var entries []*corednsv1alpha1.CoreDNSEntry
	for _, e := range list {
		logger.Info("found entry {{entry}} for {{current}}[{{zonekey}}]/{{relative}}", "entry", client.ObjectKeyFromObject(&e), "zonekey", zone.key, "current", curz, "relative", sub)
		entries = append(entries, &e)
	}

	return &Info{Zone: zone, ZoneName: curz, Entries: entries, EntryName: cur}, nil
}
