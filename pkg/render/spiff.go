package render

import (
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
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
			return nil, nil, err
		}
		result, err := ctx.Cascade(templ, nil)
		if err != nil {
			return nil, nil, err
		}
		data, err := ctx.Marshal(result)
		if err != nil {
			return nil, nil, err
		}

		manifest := result.Value().(map[string]spiffing.Node)
		if manifest["metadata"] != nil {
			metadata := manifest["metadata"]
			if metadata != nil {
				m, ok := metadata.Value().(map[string]spiffing.Node)
				if !ok {
					return nil, nil, fmt.Errorf("invalid metadata field in %s", k)
				}
				l := m["labels"]
				if l != nil {
					m, ok := l.Value().(map[string]spiffing.Node)
					if !ok {
						return nil, nil, fmt.Errorf("invalid labels field in %s", k)
					}
					if m["target"] != nil {
						s, ok := m["target"].Value().(string)
						if !ok {
							return nil, nil, fmt.Errorf("labels \"target\" must be string in %s", k)
						}
						if s == "runtime" {
							runtime[k] = data
							continue
						}
					}
				}
			}
		}
		dataplane[k] = data
	}
	return dataplane, runtime, nil
}
