package dataplane

// +kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=hostedzones/finalizers,verbs=update

// required to grant permissions anf for the entry reconciler
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
