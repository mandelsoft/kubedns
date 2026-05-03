package hostedzone

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/render"
	"github.com/mandelsoft/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestRenderManifests(logger logging.Logger) error {
	manifests, err := GetManifests()
	if err != nil {
		return err
	}

	obj := &v1alpha1.HostedZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.HostedZoneSpec{
			DomainNames: []string{
				"test.mandelsoft.org",
				"test.mandelsoft.de",
			},
		},
	}
	ctx := NewReconcileContext(context.Background(), logger, "http://api.server", "testcluster", "aws", obj)
	ctx.Simulate = true

	r := &HostedZoneReconciler{
		Options: NewOptions(),
	}

	err = handleCombi(ctx, "local", manifests, r, SERVERMODE_DATAPLANE, NewLocalMode)
	if err != nil {
		return err
	}
	err = handleCombi(ctx, "runtime server", manifests, r, SERVERMODE_RESTAPI, NewRuntimeMode)
	if err != nil {
		return err
	}
	r.Options.RuntimeNamespace = ""
	err = handleCombi(ctx, "runtime", manifests, r, SERVERMODE_DATAPLANE, NewRuntimeMode)
	if err != nil {
		return err
	}
	r.Options.RuntimeNamespace = "dns-namespace"
	err = handleCombi(ctx, "runtime namespace", manifests, r, SERVERMODE_DATAPLANE, NewRuntimeMode)
	if err != nil {
		return err
	}

	return nil
}

func handleCombi(ctx ReconcileContext, name string, manifests map[string][]byte, r *HostedZoneReconciler, s string, m ModeFactory) error {
	var err error
	r.ServerMode, err = ServerModes.Create(ctx, s, r.Options)
	if err != nil {
		return err
	}
	r.Mode = m(r)
	values, err := ctx.Values(r.Mode, false)
	if err != nil {
		return fmt.Errorf("get %s mode values: %w", name, err)
	}
	rendered, err := render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("%s mode rendering: %w", name, err)
	}
	if len(rendered.Other) > 0 {
		return fmt.Errorf("%s mode rendering: provided non-manifest", name)
	}

	fmt.Printf("***************** %s mode ******************\n", name)
	vd, _ := json.Marshal(values)
	fmt.Printf("values: %s\n", string(vd))
	fmt.Printf("dataplane manifests:\n")
	for k, v := range rendered.Dataplane {
		fmt.Printf("- %s:\n", k)
		fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
	}
	fmt.Printf("runtime manifests:\n")
	for k, v := range rendered.Runtime {
		fmt.Printf("- %s:\n", k)
		fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
	}
	return nil
}

func GetManifestsFromDir(d string) (map[string][]byte, error) {
	manifests := make(map[string][]byte)
	entries, err := fs.ReadDir(content, d)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
				data, err := content.ReadFile(d + "/" + e.Name())
				if err != nil {
					return nil, err
				}
				manifests[e.Name()] = data
			}
		}
	}
	return manifests, nil
}
