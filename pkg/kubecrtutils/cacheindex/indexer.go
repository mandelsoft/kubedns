package cacheindex

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FieldIndexer provides an indexer function for a given CEL expression.
func FieldIndexer[T client.Object](expr string) (IndexerFunc[T], error) {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Ptr {
		t = t.Elem() // Ensure we have the struct type, not the pointer
	}

	// 2. Create the environment and REGISTER the type here
	env, err := cel.NewEnv(
		// This is where you register your reflect.Type on the fly
		ext.NativeTypes(t, ext.ParseStructTag("json")),
		cel.Variable("obj", cel.DynType),
	)
	if err != nil {
		return nil, err
	}
	ast, issues := env.Compile(expr)
	if issues != nil {
		return nil, issues.Err()
	}
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
