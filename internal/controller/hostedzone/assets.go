package hostedzone

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/mandelsoft/kubedns/pkg/render"
)

//go:embed assets
var content embed.FS

const manifestsDir = "assets/manifests"

func GetManifests() (map[string][]byte, error) {
	manifests := make(map[string][]byte)
	entries, err := fs.ReadDir(content, manifestsDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
				data, err := content.ReadFile(manifestsDir + "/" + e.Name())
				if err != nil {
					return nil, err
				}
				manifests[e.Name()] = data
			}
		}
	}
	return manifests, nil
}

func PrintManifests() {
	manifests, err := GetManifests()
	if err != nil {
		panic(err)
	}
	if len(manifests) == 0 {
		fmt.Println("No manifests found")
	}
	for name := range manifests {
		fmt.Printf("- %s\n", name)
	}
}

func RenderManifests() {
	manifests, err := GetManifests()
	if err != nil {
		panic(err)
	}
	values := map[string]interface{}{
		"runtime": map[string]interface{}{
			"namespace": "dnsservice",
			// "separated": true,
		},
		"dataplane": map[string]interface{}{
			"namespace": "ns",
			// "server": "https://localhost:6443",
			// "token": "some token",
			// "cadata": "some cert",
		},
		"deployment": map[string]interface{}{
			"name":     "dns-server-ns-hz",
			"label":    "dns-service-ns-hz",
			"replicas": 3,
		},
		"service": map[string]interface{}{
			"name": "dns-server-svc-ns-hz",
		},
		"config": map[string]interface{}{
			"name": "dns-server-ns-hz",
			"zone": "hz",
		},
	}

	dataplane, runtime, err := render.Render(manifests, values)
	if err != nil {
		panic(err)
	}
	fmt.Printf("dataplane manifests:\n")
	for k, v := range dataplane {
		fmt.Printf("- %s:\n", k)
		fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
	}
	fmt.Printf("runtime manifests:\n")
	for k, v := range runtime {
		fmt.Printf("- %s:\n", k)
		fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
	}
}
