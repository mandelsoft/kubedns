package cacheindex_test

import (
	"github.com/mandelsoft/goutils/testutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cacheindex"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Field Indexer Test Environment", func() {
	ginkgo.It("field", func() {
		idx := testutils.Must(cacheindex.FieldIndexer[*corednsv1alpha1.HostedZone]("obj.spec.parentRef"))

		var obj corednsv1alpha1.HostedZone

		obj.Spec.ParentRef = "reference"
		Expect(idx(&obj)).To(Equal([]string{"reference"}))
	})
})
