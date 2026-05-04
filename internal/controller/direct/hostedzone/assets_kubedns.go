package hostedzone

import (
	"context"
	"fmt"
	"strings"

	"github.com/mandelsoft/goutils/funcs"
	"github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/render"
	"github.com/mandelsoft/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const kubeDNSDir = "assets/dns/kubedns"

func GetKubeDNSManifests() (map[string][]byte, error) {
	return GetManifestsFromDir(kubeDNSDir)
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
	r.Options.ServerMode = funcs.Must(ServerModes.Create(nil, SERVERMODE_DATAPLANE, r.Options))
	r.Mode = NewLocalMode(r)
	values, err := ctx.Values(r.Mode, false)
	if err != nil {
		return fmt.Errorf("get local mode values: %w", err)
	}
	addDNSValues(values)
	rendered, err := render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("local mode rendering: %w", err)
	}
	if len(rendered.Other) > 0 {
		return fmt.Errorf("local mode rendering: provided non-manifest")
	}

	r.Options.RuntimeNamespace = ""
	r.Options.ServerMode = funcs.Must(ServerModes.Create(nil, SERVERMODE_DATAPLANE, r.Options))
	r.Mode = NewRuntimeMode(r)
	values, err = ctx.Values(r.Mode, false)
	if err != nil {
		return fmt.Errorf("get runtime mode values: %w", err)
	}
	addDNSValues(values)
	rendered, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("runtime mode rendering: %w", err)
	}
	if len(rendered.Other) > 0 {
		return fmt.Errorf("runtime mode rendering: provided non-manifest")
	}

	r.Options.RuntimeNamespace = "dns-runtime"
	r.Options.ServerMode = funcs.Must(ServerModes.Create(nil, SERVERMODE_DATAPLANE, r.Options))
	r.Mode = NewRuntimeMode(r)
	values, err = ctx.Values(r.Mode, false)
	if err != nil {
		return fmt.Errorf("get central runtime mode values: %w", err)
	}
	addDNSValues(values)

	rendered, err = render.Render(manifests, values)
	if err != nil {
		return fmt.Errorf("central runtime mode rendering: %w", err)
	}
	if len(rendered.Other) > 0 {
		return fmt.Errorf("runtime mode rendering: provided non-manifest")
	}

	if len(rendered.Dataplane) > 0 {
		fmt.Printf("dns dataplane manifests:\n")
		for k, v := range rendered.Dataplane {
			fmt.Printf("- %s:\n", k)
			fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
		}
	}
	if len(rendered.Runtime) > 0 {
		fmt.Printf("dns runtime manifests:\n")
		for k, v := range rendered.Runtime {
			fmt.Printf("- %s:\n", k)
			fmt.Printf("    %s\n", strings.Replace(string(v), "\n", "\n    ", -1))
		}
	}
	return nil
}
