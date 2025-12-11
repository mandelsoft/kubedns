package render

import (
	"fmt"
	"strconv"

	"github.com/mandelsoft/spiff/spiffing"
	"github.com/mandelsoft/spiff/yaml"
)

func Render(manifests map[string][]byte, values map[string]interface{}) (dataplane map[string][]byte, runtime map[string][]byte, err error) {
	values = map[string]interface{}{"values": values}
	ctx, err := spiffing.New().
		WithFileSystem(nil).
		WithMode(spiffing.MODE_PRIVATE).
		WithInterpolation(true).
		WithValues(values)

	if err != nil {
		return nil, nil, err
	}
	dataplane = map[string][]byte{}
	runtime = map[string][]byte{}

	for k, v := range manifests {
		src := spiffing.NewSourceData(k, v)
		templ, err := ctx.UnmarshalSource(src)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", k, err)
		}
		result, err := ctx.Cascade(templ, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", k, err)
		}

		m, ok := result.Value().(map[string]yaml.Node)
		if !ok {
			return nil, nil, fmt.Errorf("%s: no map on root level", k)
		}
		if m["manifests"] == nil {
			if m["manifest"] == nil {
				err = retrieve(ctx, k, result, &dataplane, &runtime)
			} else {
				err = retrieve(ctx, k, result, &dataplane, &runtime)
			}
			if err != nil {
				return nil, nil, err
			}
		} else {
			switch e := m["manifests"].Value().(type) {
			case map[string]yaml.Node:
				// handle map
				for n, v := range e {
					err = retrieve(ctx, k+"::"+n, v, &dataplane, &runtime)
					if err != nil {
						return nil, nil, err
					}
				}
			case []yaml.Node:
				// handle list
				for n, v := range e {
					err = retrieve(ctx, k+"::"+strconv.Itoa(n), v, &dataplane, &runtime)
					if err != nil {
						return nil, nil, err
					}
				}
			default:
				return nil, nil, fmt.Errorf("%s: manifests must be list or map", k)
			}
		}
	}
	return dataplane, runtime, nil
}

func retrieve(ctx spiffing.Spiff, name string, result yaml.Node, dataplane *map[string][]byte, runtime *map[string][]byte) error {
	data, err := ctx.Marshal(result)
	if err != nil {
		return err
	}

	manifest, ok := result.Value().(map[string]spiffing.Node)
	if !ok {
		return fmt.Errorf("%s: no map on root level", name)
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
						(*runtime)[name] = data
						return nil
					}
				}
			}
		}
	}
	if manifest["kind"] != nil {
		(*dataplane)[name] = data
	}
	return nil
}
