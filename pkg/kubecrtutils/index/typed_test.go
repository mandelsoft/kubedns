package index_test

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

var KA = schema.GroupKind{Group: "G", Kind: "KA"}
var KB = schema.GroupKind{Group: "G", Kind: "KB"}

var AA = index.TypedObjectKey{KA, A}
var AB = index.TypedObjectKey{KA, B}
var AC = index.TypedObjectKey{KA, C}
var AD = index.TypedObjectKey{KA, D}
var AE = index.TypedObjectKey{KA, E}

var BA = index.TypedObjectKey{KB, A}
var BB = index.TypedObjectKey{KB, B}
var BC = index.TypedObjectKey{KB, C}
var BD = index.TypedObjectKey{KB, D}
var BE = index.TypedObjectKey{KB, E}

var _ = Describe("Typed Index Test Environment", func() {
	var idx index.TypedIndex

	BeforeEach(func() {
		idx = index.NewTyped()
	})

	Context("users", func() {
		Context("plain", func() {
			It("simple add", func() {
				idx.Add(R1, AA, AB)
				Expect(idx.UsersFor(R1, AB)).To(Equal(sets.New(AA)))
			})

			It("add", func() {
				idx.Add(R1, AA, AB)
				idx.Add(R1, AC, AD)
				idx.Add(R1, AE, AD)
				idx.Add(R2, AC, AB)

				Expect(idx.UsersFor(R1, AB)).To(Equal(sets.New(AA)))
				Expect(idx.UsersFor(R1, AD)).To(Equal(sets.New(AC, AE)))
				Expect(len(idx.UsersFor(R1, AC))).To(Equal(0))
				Expect(idx.UsersFor(R2, AB)).To(Equal(sets.New(AC)))

				Expect(idx.UsersForKind(R1, AB, KA)).To(Equal(sets.New(A)))
				Expect(idx.UsersForKind(R1, AD, KA)).To(Equal(sets.New(C, E)))
				Expect(len(idx.UsersForKind(R1, AC, KA))).To(Equal(0))
				Expect(idx.UsersForKind(R2, AB, KA)).To(Equal(sets.New(C)))

				Expect(len(idx.UsersForKind(R1, AB, KB))).To(Equal(0))
				Expect(len(idx.UsersForKind(R1, AD, KB))).To(Equal(0))
				Expect(len(idx.UsersForKind(R1, AC, KB))).To(Equal(0))
				Expect(len(idx.UsersForKind(R2, AB, KB))).To(Equal(0))
			})

			It("replace", func() {
				idx.Add(R1, AA, AB)
				idx.Add(R1, AC, AD)
				idx.Add(R1, AE, AD)
				idx.Add(R2, AC, AB)

				Expect(idx.UsersFor(R1, AB)).To(Equal(sets.New(AA)))
				Expect(idx.UsersForKind(R1, AB, KA)).To(Equal(sets.New(A)))
				Expect(len(idx.UsersForKind(R1, AB, KB))).To(Equal(0))

				idx.Replace(R1, AA, AC, AD)

				Expect(len(idx.UsersFor(R1, AB))).To(Equal(0))
				Expect(len(idx.UsersForKind(R1, AB, KA))).To(Equal(0))
				Expect(len(idx.UsersForKind(R1, AB, KB))).To(Equal(0))

				Expect(idx.UsersFor(R1, AC)).To(Equal(sets.New(AA)))
				Expect(idx.UsersForKind(R1, AC, KA)).To(Equal(sets.New(A)))
				Expect(len(idx.UsersForKind(R1, AC, KB))).To(Equal(0))

				Expect(idx.UsersFor(R1, AD)).To(Equal(sets.New(AA, AC, AE)))
				Expect(idx.UsersForKind(R1, AD, KA)).To(Equal(sets.New(A, C, E)))
				Expect(len(idx.UsersForKind(R1, AD, KB))).To(Equal(0))
			})

			Context("mixed", func() {
				It("simple add", func() {
					idx.Add(R1, AA, BB)
					Expect(idx.UsersFor(R1, BB)).To(Equal(sets.New(AA)))
				})

				It("add", func() {
					idx.Add(R1, AA, AB)
					idx.Add(R1, AC, AD)
					idx.Add(R1, AE, AD)
					idx.Add(R2, AC, AB)

					idx.Add(R1, AA, BC)
					idx.Add(R1, AA, BD)
					idx.Add(R1, BA, AC)
					idx.Add(R1, BA, AD)

					Expect(len(idx.UsersFor(R1, AA))).To(Equal(0))
					Expect(idx.UsersFor(R1, AB)).To(Equal(sets.New(AA)))
					Expect(idx.UsersFor(R1, AD)).To(Equal(sets.New(AC, AE, BA)))
					Expect(idx.UsersFor(R1, AC)).To(Equal(sets.New(BA)))
					Expect(idx.UsersFor(R2, AB)).To(Equal(sets.New(AC)))

					Expect(len(idx.UsersForKind(R1, AA, KA))).To(Equal(0))
					Expect(idx.UsersForKind(R1, AB, KA)).To(Equal(sets.New(A)))
					Expect(idx.UsersForKind(R1, AD, KA)).To(Equal(sets.New(C, E)))
					Expect(len(idx.UsersForKind(R1, AC, KA))).To(Equal(0))
					Expect(idx.UsersForKind(R2, AB, KA)).To(Equal(sets.New(C)))

					Expect(len(idx.UsersForKind(R1, AA, KB))).To(Equal(0))
					Expect(len(idx.UsersForKind(R1, AB, KB))).To(Equal(0))
					Expect(idx.UsersForKind(R1, AD, KB)).To(Equal(sets.New(A)))
					Expect(idx.UsersForKind(R1, AC, KB)).To(Equal(sets.New(A)))
					Expect(len(idx.UsersForKind(R2, AB, KB))).To(Equal(0))
				})
			})
		})
	})

	Context("uses", func() {
		It("add", func() {
			idx.Add(R1, AA, AB)
			idx.Add(R1, AC, AD)
			idx.Add(R1, AE, AD)
			idx.Add(R2, AC, AB)

			idx.Add(R1, AA, BC)
			idx.Add(R1, AA, BD)
			idx.Add(R1, BA, AC)
			idx.Add(R1, BA, AD)

			Expect(idx.UsedByKind(R1, AA, KA)).To(Equal(sets.New(B)))
			Expect(idx.UsedByKind(R1, AA, KB)).To(Equal(sets.New(C, D)))

			Expect(idx.UsedBy(R1, AA)).To(Equal(sets.New(AB, BC, BD)))
			Expect(len(idx.UsedBy(R1, AB))).To(Equal(0))
			Expect(idx.UsedBy(R1, AC)).To(Equal(sets.New(AD)))
			Expect(len(idx.UsedBy(R1, AD))).To(Equal(0))
			Expect(idx.UsedBy(R1, AE)).To(Equal(sets.New(AD)))

			Expect(len(idx.UsedBy(R2, AA))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, AB))).To(Equal(0))
			Expect(idx.UsedBy(R2, AC)).To(Equal(sets.New(AB)))
			Expect(len(idx.UsedBy(R2, AD))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, AE))).To(Equal(0))

			Expect(idx.UsedBy(R1, BA)).To(Equal(sets.New(AC, AD)))
			Expect(len(idx.UsedBy(R1, BB))).To(Equal(0))
			Expect(len(idx.UsedBy(R1, BC))).To(Equal(0))
			Expect(len(idx.UsedBy(R1, BD))).To(Equal(0))
			Expect(len(idx.UsedBy(R1, BE))).To(Equal(0))

			Expect(len(idx.UsedBy(R2, BA))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, BB))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, BC))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, BD))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, BE))).To(Equal(0))

		})
	})
})
