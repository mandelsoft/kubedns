package index_test

import (
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const R1 = "relation1"
const R2 = "relation2"

var A = client.ObjectKey{Namespace: "default", Name: "A"}
var B = client.ObjectKey{Namespace: "default", Name: "B"}
var C = client.ObjectKey{Namespace: "default", Name: "C"}
var D = client.ObjectKey{Namespace: "default", Name: "D"}
var E = client.ObjectKey{Namespace: "default", Name: "E"}

var _ = Describe("Untyped Index Test Environment", func() {
	var idx index.UntypedIndex

	BeforeEach(func() {
		idx = index.NewUntyped()
	})

	Context("users", func() {
		It("add", func() {
			idx.Add(R1, A, B)
			idx.Add(R1, C, D)
			idx.Add(R1, E, D)
			idx.Add(R2, C, B)
			Expect(idx.UsersFor(R1, B)).To(Equal(sets.New(A)))
			Expect(idx.UsersFor(R1, D)).To(Equal(sets.New(C, E)))
			Expect(len(idx.UsersFor(R1, C))).To(Equal(0))
			Expect(idx.UsersFor(R2, B)).To(Equal(sets.New(C)))
		})

		It("replace", func() {
			idx.Add(R1, A, B)
			idx.Add(R1, C, D)
			idx.Add(R1, E, D)
			idx.Add(R2, C, B)

			Expect(idx.UsersFor(R1, B)).To(Equal(sets.New(A)))
			idx.Replace(R1, A, C, D)
			Expect(len(idx.UsersFor(R1, B))).To(Equal(0))
			Expect(idx.UsersFor(R1, C)).To(Equal(sets.New(A)))
			Expect(idx.UsersFor(R1, D)).To(Equal(sets.New(A, C, E)))
		})
	})

	Context("uses", func() {
		It("add", func() {
			idx.Add(R1, A, B)
			idx.Add(R1, C, D)
			idx.Add(R1, E, D)
			idx.Add(R2, C, B)

			Expect(idx.UsedBy(R1, A)).To(Equal(sets.New(B)))
			Expect(len(idx.UsedBy(R1, B))).To(Equal(0))
			Expect(idx.UsedBy(R1, C)).To(Equal(sets.New(D)))
			Expect(len(idx.UsedBy(R1, D))).To(Equal(0))
			Expect(idx.UsedBy(R1, E)).To(Equal(sets.New(D)))

			Expect(len(idx.UsedBy(R2, A))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, B))).To(Equal(0))
			Expect(idx.UsedBy(R2, C)).To(Equal(sets.New(B)))
			Expect(len(idx.UsedBy(R2, D))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, E))).To(Equal(0))
		})

		It("replace", func() {
			idx.Add(R1, A, B)
			idx.Add(R1, C, D)
			idx.Add(R1, E, D)
			idx.Add(R2, C, B)

			idx.Replace(R1, A, C, D)

			Expect(idx.UsedBy(R1, A)).To(Equal(sets.New(C, D)))
			Expect(len(idx.UsedBy(R1, B))).To(Equal(0))
			Expect(idx.UsedBy(R1, C)).To(Equal(sets.New(D)))
			Expect(len(idx.UsedBy(R1, D))).To(Equal(0))
			Expect(idx.UsedBy(R1, E)).To(Equal(sets.New(D)))

			Expect(len(idx.UsedBy(R2, A))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, B))).To(Equal(0))
			Expect(idx.UsedBy(R2, C)).To(Equal(sets.New(B)))
			Expect(len(idx.UsedBy(R2, D))).To(Equal(0))
			Expect(len(idx.UsedBy(R2, E))).To(Equal(0))

		})
	})
})
