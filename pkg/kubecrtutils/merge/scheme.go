package merge

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi3"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

type ObjectMerger struct {
	converter   managedfields.TypeConverter
	managerName string
}

func NewObjectMerger(c managedfields.TypeConverter, managerName string) (*ObjectMerger, error) {
	return &ObjectMerger{converter: c, managerName: managerName}, nil
}

func NewConverterV3(config *rest.Config) (managedfields.TypeConverter, error) {
	// 1. Setup discovery and the V3 Root
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	v3Client := dc.OpenAPIV3()

	// openapi3.NewRoot is the official helper to traverse the V3 discovery
	root := openapi3.NewRoot(v3Client)

	definitions := make(map[string]*spec.Schema)

	// 2. Fetch the "Root" which contains all GroupVersions
	paths, err := v3Client.Paths()
	if err != nil {
		return nil, err
	}

	for gvStr, _ := range paths {
		// Use the root helper to get the spec for each GroupVersion
		// This handles the network requests to /openapi/v3/apis/...
		if strings.HasPrefix(gvStr, "apis/") {
			gvStr = strings.TrimPrefix(gvStr, "apis/")
		} else if strings.HasPrefix(gvStr, "api/") {
			gvStr = strings.TrimPrefix(gvStr, "api/")
		}
		gv, err := schema.ParseGroupVersion(gvStr)
		if err != nil {
			continue
		}
		spec3, err := root.GVSpec(gv)
		if err != nil {
			continue
		}

		// Convert OpenAPI V3 components to the V2 spec.Schema map for TypeConverter
		for name, s := range spec3.Components.Schemas {
			definitions[name] = s
		}
	}

	return managedfields.NewTypeConverter(definitions, false)
}
