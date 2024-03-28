package restc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

type TypeResolvingContext struct {
	projectRoot string
	module      string
	packagePath string
	imports     map[string]string
}

func NewTypeResolvingContext(projectRoot, module, packageName string, imports []*ast.ImportSpec) TypeResolvingContext {
	trc := TypeResolvingContext{
		projectRoot: projectRoot,
		module:      module,
		packagePath: packageName,
		imports:     make(map[string]string),
	}

	for _, is := range imports {
		pv := strings.ReplaceAll(is.Path.Value, "\"", "")
		if is.Name != nil {
			trc.imports[is.Name.Name] = pv
		} else {
			pathParts := strings.Split(pv, "/")
			trc.imports[pathParts[len(pathParts)-1]] = pv
		}
	}

	return trc
}

type TypeResolver struct {
	// full/path/to/package TypeName
	resolvedTypes map[string]*ResolvedType
}

type ResolvedType struct {
	PackageName      string
	File             string
	Name             string
	Doc              *ast.CommentGroup
	Type             ast.Node
	ResolvingContext *TypeResolvingContext
}

func NewTypeResolver() TypeResolver {
	return TypeResolver{
		resolvedTypes: make(map[string]*ResolvedType),
	}
}

// ResolveIdentifierExpr resolves type expression into full qualified type name (with package and without package alias)
//
// format: full/path/to/package TypeName
func (r *TypeResolver) ResolveIdentifierExpr(trctx TypeResolvingContext, expr ast.Expr) string {
	switch i := expr.(type) {
	case *ast.Ident:
		if IsPrimitive(i.Name) {
			return i.Name
		}
		return trctx.packagePath + " " + i.Name
	case *ast.SelectorExpr:
		packageAlias := i.X.(*ast.Ident).Name
		packagePath := trctx.imports[packageAlias]
		return strings.TrimSpace(packagePath + " " + i.Sel.Name)
	// case *ast.ArrayType:
	// 	switch el := i.Elt.(type) {
	// 	case *ast.Ident:
	// 	case *ast.SelectorExpr:
	// 		packageAlias := el.X.(*ast.Ident).Name
	// 		packagePath := trctx.imports[packageAlias]
	// 		return strings.TrimSpace(packagePath + " []" + el.Sel.Name)
	// 	}
	// 	return ""
	default:
		panic("unsupported ast node to resolve identifier")
	}
}

func (r *TypeResolver) ResolveType(trctx TypeResolvingContext, identifier string) (*ResolvedType, error) {
	var packageIdentifier string
	var typeName string

	parts := strings.Split(identifier, " ")
	if len(parts) == 2 {
		packageIdentifier = parts[0]
		typeName = parts[1]
	} else {
		packageIdentifier = trctx.packagePath
		typeName = parts[0]
	}

	normalizedIdentifier := packageIdentifier + " " + typeName

	if rt, ok := r.resolvedTypes[normalizedIdentifier]; ok {
		return rt, nil
	}

	var rt *ResolvedType

	if strings.HasPrefix(packageIdentifier, trctx.module) {
		packagePath := filepath.Join(trctx.projectRoot, strings.TrimPrefix(strings.TrimPrefix(packageIdentifier, trctx.module), "/"))

		files, err := filepath.Glob(packagePath + "/*.go")
		if err != nil {
			return nil, err
		}

		for _, fn := range files {
			fast, err := parser.ParseFile(token.NewFileSet(), fn, nil, parser.ParseComments)
			if err != nil {
				return nil, fmt.Errorf("resolve type parse error: %w", err)
			}

			ast.Inspect(fast, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.File:
					return true
				case *ast.GenDecl:
					for _, s := range node.Specs {
						switch ts := s.(type) {
						case *ast.TypeSpec:
							r.resolvedTypes[packageIdentifier+" "+ts.Name.Name] = &ResolvedType{
								Name:        ts.Name.Name,
								File:        fn,
								PackageName: packageIdentifier,
								Doc:         node.Doc,
								Type:        ts.Type,
							}
						}
					}
				}
				return false
			})
		}

		if t, ok := r.resolvedTypes[normalizedIdentifier]; ok {
			rt = t
		} else {
			r.resolvedTypes[normalizedIdentifier] = nil
		}

	} else {
		// TODO: resolve external packages
		// do it like PATH
		// look at go.mod for dependencies
		return nil, fmt.Errorf("external packages is not supported currently")
	}

	return rt, nil
}

func (r *TypeResolver) AnalyzePackageFileAst(packageIdentifier string, fileAst *ast.File) {
	ast.Inspect(fileAst, func(n ast.Node) bool {
		switch concreteNode := n.(type) {
		case *ast.GenDecl:
			fmt.Println(concreteNode.Doc)
		case *ast.FuncDecl:
			fmt.Println(concreteNode.Doc)
		}
		return false
	})
	// TODO: analyze package
}
