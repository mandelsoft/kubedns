package runtime

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces;configmaps;secrets;services,verbs=get;list;watch;create;update;patch;delete
