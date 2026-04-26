package zonemodel_test

import (
	"context"
	"slices"

	"github.com/mandelsoft/goutils/sliceutils"
	. "github.com/mandelsoft/goutils/testutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/logging"
	"github.com/mandelsoft/logging/logrusl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Index struct {
	entries map[zonemodel.ZoneKey]map[string][]corednsv1alpha1.CoreDNSEntry
}

var _ zonemodel.Index = (*Index)(nil)

func NewIndex() *Index {
	return &Index{entries: make(map[zonemodel.ZoneKey]map[string][]corednsv1alpha1.CoreDNSEntry)}
}

func (i *Index) LookupRelativeDomainName(ctx context.Context, zone zonemodel.ZoneKey, rel string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	zoneentries := i.entries[zone]
	if zoneentries == nil {
		return nil, nil
	}
	return zoneentries[rel], nil
}

func (i *Index) LookupIP(ctx context.Context, zone zonemodel.ZoneKey, ip string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	zoneentries := i.entries[zone]
	if zoneentries == nil {
		return nil, nil
	}
	var result []corednsv1alpha1.CoreDNSEntry
	for _, list := range zoneentries {
		for _, e := range list {
			if slices.Contains(e.Spec.A, ip) {
				result = append(result, e)
			}
		}
	}
	return result, nil
}

func (i *Index) AddEntry(s zonemodel.Source, entry *corednsv1alpha1.CoreDNSEntry) {
	zonekey := zonemodel.ZoneRefKeyFromObject(s.GetName(), entry, entry.Spec.ZoneRef)
	zoneentries := i.entries[zonekey]
	if zoneentries == nil {
		zoneentries = make(map[string][]corednsv1alpha1.CoreDNSEntry)
		i.entries[zonekey] = zoneentries
	}
	for _, n := range entry.Spec.DNSNames {
		n = zonemodel.Rdn(n)
		zoneentries[n] = append(zoneentries[n], *entry)
	}
}

////////////////////////////////////////////////////////////////////////////////

type source struct {
	name string
}

var _ zonemodel.Source = (*source)(nil)

func (s *source) GetName() string {
	return s.name
}

////////////////////////////////////////////////////////////////////////////////

var ZoneA *corednsv1alpha1.HostedZone
var ZoneAKey zonemodel.ZoneKey

var EntryA_demo *corednsv1alpha1.CoreDNSEntry
var EntryA_test *corednsv1alpha1.CoreDNSEntry
var EntryA_NS *corednsv1alpha1.CoreDNSEntry
var ZoneAA *corednsv1alpha1.HostedZone

var ZoneAAKey zonemodel.ZoneKey
var Source = &source{"cluster"}

var EntryAA_demo *corednsv1alpha1.CoreDNSEntry

func init() {
	ZoneA = &corednsv1alpha1.HostedZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.HostedZoneSpec{
			DomainNames: []string{"mandelsoft.de", "mandelsoft.org"},
		},
	}
	ZoneAKey = zonemodel.ZoneKeyFromObject(Source.GetName(), ZoneA)

	ZoneAA = &corednsv1alpha1.HostedZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aa",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.HostedZoneSpec{
			DomainNames: []string{"demo", "test"},
			ParentRef:   "a",
		},
	}
	ZoneAAKey = zonemodel.ZoneKeyFromObject(Source.GetName(), ZoneAA)

	EntryA_demo = &corednsv1alpha1.CoreDNSEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.CoreDNSSpec{
			DNSNames: []string{"demo.test", "demo.test2"},
			ZoneRef:  "a",
			A:        sliceutils.AsSlice("8.8.8.8"),
		},
	}

	EntryA_test = &corednsv1alpha1.CoreDNSEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.CoreDNSSpec{
			DNSNames: []string{"test"},
			ZoneRef:  "a",
			A:        sliceutils.AsSlice("8.8.8.9"),
		},
	}

	EntryA_NS = &corednsv1alpha1.CoreDNSEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a_ns",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.CoreDNSSpec{
			DNSNames: []string{"nested.test"},
			NS:       []string{"nested.test.com"},
			ZoneRef:  "a",
		},
	}

	EntryAA_demo = &corednsv1alpha1.CoreDNSEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aa-demo",
			Namespace: "default",
		},
		Spec: corednsv1alpha1.CoreDNSSpec{
			DNSNames: []string{"demo"},
			ZoneRef:  "aa",
			A:        sliceutils.AsSlice("9.9.9.9"),
		},
	}

}

var _ = Describe("Test Environment", func() {

	var logger logging.Logger
	var model *zonemodel.Model
	var index *Index
	var zoneA *zonemodel.Zone

	BeforeEach(func() {
		ctx := logrusl.Human().New()
		ctx.SetDefaultLevel(logging.InfoLevel)
		logger = ctx.Logger(logging.NewRealm("test"))
		index = NewIndex()
		model = zonemodel.New(index)
		zoneA = model.AddZone(Source, ZoneAKey, ZoneA)
	})

	Context("fqdn to record", func() {
		It("simple entry access", func() {
			index.AddEntry(Source, EntryA_test)

			Expect(zoneA.ResolveFQDN(logger, "test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneA,
				ZoneNames:  sliceutils.AsSlice("mandelsoft.de."),
				Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryA_test},
				EntryNames: sliceutils.AsSlice("test.mandelsoft.de."),
			}))
		})

		It("simple entry access", func() {
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryA_demo)

			Expect(zoneA.ResolveFQDN(logger, "test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneA,
				ZoneNames:  sliceutils.AsSlice("mandelsoft.de."),
				Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryA_test},
				EntryNames: sliceutils.AsSlice("test.mandelsoft.de."),
			}))
		})

		It("simple alternate entry access", func() {
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryA_demo)

			Expect(zoneA.ResolveFQDN(logger, "demo.test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneA,
				ZoneNames:  sliceutils.AsSlice("mandelsoft.de."),
				Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryA_demo},
				EntryNames: sliceutils.AsSlice("demo.test.mandelsoft.de."),
			}))
		})

		It("ignore forwarded", func() {
			zoneAA := model.AddZone(Source, ZoneAAKey, ZoneAA)
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryA_demo)

			Expect(zoneA.ResolveFQDN(logger, "demo.test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneAA,
				ZoneNames:  sliceutils.AsSlice("test.mandelsoft.de."),
				Entries:    nil,
				EntryNames: sliceutils.AsSlice("demo.test.mandelsoft.de."),
			}))
		})

		It("use forwarded", func() {
			zoneAA := model.AddZone(Source, ZoneAAKey, ZoneAA)
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryAA_demo)

			Expect(zoneA.ResolveFQDN(logger, "demo.test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneAA,
				ZoneNames:  sliceutils.AsSlice("test.mandelsoft.de."),
				Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryAA_demo},
				EntryNames: sliceutils.AsSlice("demo.test.mandelsoft.de."),
			}))
		})

		It("provide forwarded NS", func() {
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryA_NS)

			Expect(zoneA.ResolveFQDN(logger, "demo.nested.test.mandelsoft.de")).To(DeepEqual(&zonemodel.Info{
				Zone:       zoneA,
				ZoneNames:  sliceutils.AsSlice("mandelsoft.de."),
				Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryA_NS},
				EntryNames: sliceutils.AsSlice("nested.test.mandelsoft.de."),
			}))
		})
	})

	Context("ip to record", func() {
		It("simple entry access", func() {
			index.AddEntry(Source, EntryA_test)

			Expect(zoneA.ResolveIP(logger, "8.8.8.9")).To(DeepEqual([]*zonemodel.Info{
				{
					Zone:       zoneA,
					ZoneNames:  sliceutils.AsSlice("mandelsoft.de.", "mandelsoft.org."),
					Entries:    []*corednsv1alpha1.CoreDNSEntry{EntryA_test},
					EntryNames: sliceutils.AsSlice("test.mandelsoft.de.", "test.mandelsoft.org."),
				},
			}))
		})

		FIt("forwarded entry access", func() {
			zoneAA := model.AddZone(Source, ZoneAAKey, ZoneAA)
			index.AddEntry(Source, EntryA_test)
			index.AddEntry(Source, EntryAA_demo)

			Expect(zoneA.ResolveIP(logger, "9.9.9.9")).To(DeepEqual([]*zonemodel.Info{
				{
					Zone: zoneAA,
					ZoneNames: sliceutils.AsSlice(
						"demo.mandelsoft.de.", "demo.mandelsoft.org.",
						"test.mandelsoft.de.", "test.mandelsoft.org."),
					Entries: []*corednsv1alpha1.CoreDNSEntry{EntryAA_demo},
					EntryNames: sliceutils.AsSlice(
						"demo.demo.mandelsoft.de.", "demo.demo.mandelsoft.org.",
						"demo.test.mandelsoft.de.", "demo.test.mandelsoft.org."),
				},
			}))

		})
	})
})
