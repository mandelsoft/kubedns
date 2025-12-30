package hostedzone

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/mandelsoft/kubedns/pkg/render"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestRenderManifests() error {
	manifests, err := GetManifests()
	if err != nil {
		return err
	}
	ctx := NewReconcileContext(context.Background(), Log, "http://api.server", "aws", client.ObjectKey{Name: "myzone", Namespace: "default"})
	ctx.Simulate = true

	r := &HostedZoneReconciler{
		Options: NewOptions(),
	}
	values, err := ctx.Values(NewLocalMode(r), false)
	if err != nil {
		return fmt.Errorf("get local mode values: %w", err)
	}
	_, _, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("local mode rendering: %w", err)
	}

	r.Options.RuntimeNamespace = ""
	values, err = ctx.Values(NewRuntimeMode(r), false)
	if err != nil {
		return fmt.Errorf("get runtime mode values: %w", err)
	}
	dataplane, runtime, err := render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("runtime mode rendering: %w", err)
	}

	r.Options.RuntimeNamespace = "dns-runtime"
	values, err = ctx.Values(NewRuntimeMode(r), false)
	if err != nil {
		return fmt.Errorf("get central runtime mode values: %w", err)
	}
	dataplane, runtime, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("cenbtral runtime mode rendering: %w", err)
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
	return nil
}
