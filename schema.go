package restc

import (
	"fmt"
	"go/ast"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Schema struct {
	Ref string

	Type  string
	Items *Schema

	Required   []string
	Properties *orderedmap.OrderedMap[string, *Schema]
}

func NewSchemaFromNode(node ast.Expr) *Schema {
	switch n := node.(type) {
	case *ast.Ident:
		// TODO: primitive or another type (make ref to another type instead of embedding)
	case *ast.StructType:
		// TODO: struct
		for _, field := range n.Fields.List {
			s := NewSchemaFromNode(field.Type)
			fmt.Println(field, s)
		}
	case *ast.ArrayType:
		// TODO: array
	case *ast.MapType:
		// TODO: map
	}
	return nil
}
