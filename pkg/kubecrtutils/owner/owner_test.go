package owner_test

import (
	"github.com/mandelsoft/goutils/generics"
	. "github.com/mandelsoft/goutils/testutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	sigclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var _ = Describe("Owner Test Environment", func() {
	var _owner sigclient.Object
	var _slaveDefault sigclient.Object
	var _slaveOther sigclient.Object
	var gvkOwner schema.GroupVersionKind
	var gvkSlave schema.GroupVersionKind

	BeforeEach(func() {
		_owner = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "owner",
			},
		}

		_slaveDefault = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "secret",
			},
		}

		_slaveOther = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "secret",
			},
		}

		gvkOwner, _ = apiutil.GVKForObject(_owner, clientgoscheme.Scheme)
		gvkSlave, _ = apiutil.GVKForObject(_slaveDefault, clientgoscheme.Scheme)
	})

	Context("local", func() {
		handler := owner.ForNames(clientgoscheme.Scheme, "", "")

		It("same namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveDefault))
			Expect(handler.GetOwner(_slaveDefault, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
			Expect(handler.GetOwner(_slaveDefault, gvkSlave.GroupKind())).To(BeNil())
		})

		It("cross namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveOther))
			Expect(handler.GetOwner(_slaveOther, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
			Expect(handler.GetOwner(_slaveOther, gvkSlave.GroupKind())).To(BeNil())
		})
	})

	Context("remote", func() {
		handler := owner.ForNames(clientgoscheme.Scheme, "cluster", "")

		It("same namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveDefault))
			Expect(handler.GetOwner(_slaveDefault, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
			Expect(handler.GetOwner(_slaveDefault, gvkSlave.GroupKind())).To(BeNil())
		})

		It("cross namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveOther))
			Expect(handler.GetOwner(_slaveOther, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
			Expect(handler.GetOwner(_slaveOther, gvkSlave.GroupKind())).To(BeNil())
		})
	})

	Context("combined", func() {
		other := owner.ForNames(clientgoscheme.Scheme, "other", "")
		handler := owner.ForNames(clientgoscheme.Scheme, "cluster", "")

		BeforeEach(func() {
			MustBeSuccessful(other.SetOwner(_owner, _slaveDefault))
		})

		It("not found", func() {
			Expect(handler.GetOwner(_slaveDefault, gvkOwner.GroupKind())).To(BeNil())
			Expect(other.GetOwner(_slaveDefault, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
		})

		It("same namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveDefault))
			Expect(handler.GetOwner(_slaveDefault, gvkSlave.GroupKind())).To(BeNil())
		})

		It("cross namespace", func() {
			MustBeSuccessful(handler.SetOwner(_owner, _slaveOther))
			Expect(handler.GetOwner(_slaveOther, gvkOwner.GroupKind())).To(Equal(generics.PointerTo(sigclient.ObjectKeyFromObject(_owner))))
			Expect(handler.GetOwner(_slaveOther, gvkSlave.GroupKind())).To(BeNil())
		})
	})
})
