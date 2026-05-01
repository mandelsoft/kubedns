package render

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	yaml2 "github.com/goccy/go-yaml"
	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/spiff/spiffing"
	"github.com/mandelsoft/spiff/yaml"
)

type Rendered struct {
	Dataplane map[string][]byte
	Runtime   map[string][]byte
	Other     map[string][]byte
}

func newRendered() *Rendered {
	return &Rendered{
		Dataplane: make(map[string][]byte),
		Runtime:   make(map[string][]byte),
		Other:     make(map[string][]byte),
	}
}

func Render(manifests map[string][]byte, values map[string]interface{}, key ...string) (rendered *Rendered, err error) {
	values = map[string]interface{}{"values": values}
	ctx, err := spiffing.New().
		WithFileSystem(nil).
		WithMode(spiffing.MODE_PRIVATE).
		WithInterpolation(true).
		WithValues(values)

	if err != nil {
		return nil, err
	}
	rendered = newRendered()

	for k, v := range manifests {
		src := spiffing.NewSourceData(k, v)
		templ, err := ctx.UnmarshalSource(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		result, err := ctx.Cascade(templ, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		m, ok := result.Value().(map[string]yaml.Node)
		if !ok {
			return nil, fmt.Errorf("%s: no map on root level", k)
		}
		if len(key) > 0 {
			err = retrieve(ctx, k, m[key[0]], rendered)
		} else if m["manifests"] == nil {
			if m["manifest"] == nil {
				err = retrieve(ctx, k, result, rendered)
			} else {
				err = retrieve(ctx, k, m["manifest"], rendered)
			}
			if err != nil {
				return nil, err
			}
		} else {
			switch e := m["manifests"].Value().(type) {
			case map[string]yaml.Node:
				// handle map
				for n, v := range e {
					err = retrieve(ctx, k+"::"+n, v, rendered)
					if err != nil {
						return nil, err
					}
				}
			case []yaml.Node:
				// handle list
				for n, v := range e {
					err = retrieve(ctx, k+"::"+strconv.Itoa(n), v, rendered)
					if err != nil {
						return nil, err
					}
				}
			default:
				return nil, fmt.Errorf("%s: manifests must be list or map", k)
			}
		}
	}
	apply_hashes(rendered.Runtime)
	return rendered, nil
}

func apply_hashes(manifests map[string][]byte) error {
	mod := true

	for mod {
		mod = false
		for n, v := range manifests {
			var m map[string]interface{}

			err := yaml2.Unmarshal(v, &m)
			if err != nil {
				return err
			}

			meta := generics.Cast[map[string]interface{}](m["metadata"])
			if meta != nil {
				annos := generics.Cast[map[string]interface{}](meta["annotations"])
				changed := false
				for k, cur := range annos {
					if strings.HasPrefix(k, "hashes.") {
						data := manifests[k[len("hashes."):]]
						if len(data) > 0 {
							h := sha256.Sum256(data)
							n := hex.EncodeToString(h[:])
							if n != cur {
								changed = true
								annos[k] = n
							}
						} else {
							changed = true
							delete(annos, k)
						}
					}
				}
				if changed {
					mod = true
					v, err := yaml2.Marshal(m)
					if err != nil {
						return err
					}
					manifests[n] = v
				}
			}
		}
	}
	return nil
}

func retrieve(ctx spiffing.Spiff, name string, result yaml.Node, rendered *Rendered) error {
	data, err := ctx.Marshal(result)
	if err != nil {
		return err
	}

	manifest, ok := result.Value().(map[string]spiffing.Node)
	if ok {
		if len(manifest) == 0 {
			return nil
		}
		if manifest["metadata"] != nil {
			metadata := manifest["metadata"]
			if metadata != nil {
				m, ok := metadata.Value().(map[string]spiffing.Node)
				if !ok {
					return fmt.Errorf("%s: invalid metadata field", name)
				}
				l := m["labels"]
				if l != nil {
					m, ok := l.Value().(map[string]spiffing.Node)
					if !ok {
						return fmt.Errorf("%s: invalid labels field", name)
					}
					if m["target"] != nil {
						s, ok := m["target"].Value().(string)
						if !ok {
							return fmt.Errorf("%s: labels \"target\" must be string", name)
						}
						if s == "runtime" {
							rendered.Runtime[name] = data
							return nil
						}
					}
				}
			}
		}
		if manifest["kind"] != nil {
			rendered.Dataplane[name] = data
		} else {
			rendered.Other[name] = data
		}
	} else {
		rendered.Other[name] = data
	}
	return nil
}
