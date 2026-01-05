package cacheindex

import (
	"github.com/google/cel-go/cel"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FieldIndexer provides an indexer function for a given CEL expression.
func FieldIndexer[T client.Object](expr string) (IndexerFunc[T], error) {
	env, _ := cel.NewEnv(cel.Variable("obj", cel.DynType))

	ast, _ := env.Compile(`obj.Spec.Template.Spec.Containers[0].Name`)
	program, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	return func(obj T) []string {
		out, _, err := program.Eval(map[string]any{"obj": obj})
		if err != nil {
			return nil
		}
		switch v := out.Value().(type) {
		case string:
			return []string{v}
		case []string:
			return v
		default:
			return nil
		}
	}, nil
}
