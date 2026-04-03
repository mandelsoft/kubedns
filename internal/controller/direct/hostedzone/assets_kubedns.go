package hostedzone

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/mandelsoft/kubedns/pkg/render"
	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const kubeDNSDir = "assets/dns/kubedns"

func GetKubeDNSManifests() (map[string][]byte, error) {
	manifests := make(map[string][]byte)
	entries, err := fs.ReadDir(content, kubeDNSDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
				data, err := content.ReadFile(kubeDNSDir + "/" + e.Name())
				if err != nil {
					return nil, err
				}
				manifests[e.Name()] = data
			}
		}
	}
	return manifests, nil
}

func addDNSValues(values map[string]interface{}) {
	values["dns"] = map[string]interface{}{
		"name":      "dns",
		"namespace": "dns-system",
		"class":     "dns-class",
		"ips": []interface{}{
			"1.2.3.4",
		},
		"cnames": []interface{}{
			"alice.bob",
		},
		"dnsnames": []interface{}{
			"dns.nameserver.bob",
		},
	}
}

func TestRenderKubeDNSManifests(logger logging.Logger) error {
	manifests, err := GetKubeDNSManifests()
	if err != nil {
		return err
	}
	ctx := NewReconcileContext(context.Background(), logger, "http://api.server", "testcluster", "aws", client.ObjectKey{Name: "myzone", Namespace: "default"})
	ctx.Simulate = true

	r := &HostedZoneReconciler{
		Options: NewOptions(),
	}
	values, err := ctx.Values(NewLocalMode(r), false)
	if err != nil {
		return fmt.Errorf("get local mode values: %w", err)
	}
	addDNSValues(values)
	_, _, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("local mode rendering: %w", err)
	}

	r.Options.RuntimeNamespace = ""
	values, err = ctx.Values(NewRuntimeMode(r), false)
	if err != nil {
		return fmt.Errorf("get runtime mode values: %w", err)
	}
	addDNSValues(values)
	dataplane, runtime, err := render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("runtime mode rendering: %w", err)
	}

	r.Options.RuntimeNamespace = "dns-runtime"
	values, err = ctx.Values(NewRuntimeMode(r), false)
	if err != nil {
		return fmt.Errorf("get central runtime mode values: %w", err)
	}
	addDNSValues(values)

	dataplane, runtime, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("central runtime mode rendering: %w", err)
	}

	if len(dataplane) > 0 {
		fmt.Printf("dns dataplane manifests:\n")
		for k, v := range dataplane {
			fmt.Printf("- %s:\n", k)
			fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
		}
	}
	if len(runtime) > 0 {
		fmt.Printf("dns runtime manifests:\n")
		for k, v := range runtime {
			fmt.Printf("- %s:\n", k)
			fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
		}
	}
	return nil
}
