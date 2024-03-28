package restc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var pathParamRegex *regexp.Regexp = regexp.MustCompile(`\{([[:alnum:]]*)\}`)

type RestCompilerAnalyzer struct {
	logger      *slog.Logger
	moduleName  string
	projectRoot string
	filePattern *regexp.Regexp
	resolver    TypeResolver

	Definitions Definitions
}

func NewRestCompilerAnalyzer(logger *slog.Logger, moduleName, projectRoot string, filePattern *regexp.Regexp) RestCompilerAnalyzer {
	return RestCompilerAnalyzer{
		logger:      logger,
		moduleName:  moduleName,
		projectRoot: projectRoot,
		filePattern: filePattern,
		resolver:    NewTypeResolver(),
		Definitions: NewDefinitions(),
	}
}

func (r *RestCompilerAnalyzer) Analyze() {
	filepath.WalkDir(r.projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			r.logger.Error("walk dir error", "path", path, "error", err)
			os.Exit(1)
		}

		modulePath := strings.TrimPrefix(path, r.projectRoot+"/")

		if !r.filePattern.MatchString(modulePath) {
			return nil
		}

		fast, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			r.logger.Error("cannot parse file", "file", path, "error", err)
			os.Exit(1)
		}

		fileName := filepath.Base(modulePath)
		packagePath := r.moduleName + "/" + filepath.Dir(modulePath)

		r.AnalyzeFile(fast, packagePath, fileName)

		return nil
	})

	packageAliases := make(map[string]string, 0)

	reg := regexp.MustCompile(`[\/\.\-]+`)

	for ti, ts := range r.Definitions.Types {
		parts := strings.Split(ti, " ")
		packageIdentifier := parts[0]
		alias := strings.ToLower(reg.ReplaceAllString(packageIdentifier, "_"))
		packageAliases[packageIdentifier] = alias
		ts.Alias = alias + "." + parts[1]
		r.Definitions.Types[ti] = ts
	}

	for name, c := range r.Definitions.Controllers {
		alias := strings.ToLower(reg.ReplaceAllString(c.Package, "_"))
		packageAliases[c.Package] = alias
		c.Alias = alias + "." + c.Name
		r.Definitions.Controllers[name] = c
	}

	imports := make([]string, 0)
	for p, a := range packageAliases {
		imports = append(imports, a+` `+`"`+p+`"`)
	}

	r.Definitions.Imports = imports

}

func (r *RestCompilerAnalyzer) AnalyzeFile(fast *ast.File, packagePath, fileName string) {

	// responders

	var trctx TypeResolvingContext

	ast.Inspect(fast, func(n ast.Node) bool {
		switch node := n.(type) {
		// parse imports
		case *ast.File:
			trctx = NewTypeResolvingContext(r.projectRoot, r.moduleName, packagePath, node.Imports)
			return true
		// Find controllers
		case *ast.GenDecl:
			for _, s := range node.Specs {
				switch ts := s.(type) {
				case *ast.TypeSpec:
					switch ts.Type.(type) {
					// case *ast.InterfaceType:
					// definedTypes[ts.Name.Name] = its
					case *ast.StructType:
						// Find controllers
						annotations := ParseAnnotations(node.Doc)
						if controllerAnnotation, ok := annotations["@Controller"]; ok {
							basePath := ""

							if len(controllerAnnotation) > 0 {
								basePath = controllerAnnotation[0]
							}

							r.Definitions.Controllers[ts.Name.Name] = Controller{
								Package:   packagePath,
								File:      path.Join(packagePath, fileName),
								Name:      ts.Name.Name,
								Base:      basePath,
								Resources: make(map[string]Resource),
							}
						}
					}
				}
			}
		// Find declared resources
		case *ast.FuncDecl:
			annotations := ParseAnnotations(node.Doc)
			if _, ok := annotations["@Resource"]; !ok {
				return false
			}

			resourceAnnotation := strings.Split(annotations["@Resource"][0], " ")
			if len(resourceAnnotation) < 2 {
				r.logger.Error("incorrect resource annotation", "func", node.Name.Name)
				return false
			}

			method := resourceAnnotation[0]
			pathPattern := resourceAnnotation[1]

			// paramsAnnotations := make(map[string]string, 0)
			// for _, pa := range annotations["@Param"] {
			// 	parts := strings.SplitN(pa, " ", 2)
			// 	if len(parts) == 2 {
			// 		paramsAnnotations[parts[0]] = parts[1]
			// 	} else {
			// 		paramsAnnotations[parts[0]] = ""
			// 	}
			// }

			// build params
			params := make([]Parameter, 0)

			// find query and body params
			if node.Type.Params.NumFields() > 1 {
				for _, field := range node.Type.Params.List {
					for _, fieldName := range field.Names {
						paramTypeIdent := r.resolver.ResolveIdentifierExpr(trctx, field.Type)

						if paramTypeIdent == "context Context" {
							params = append(params, Parameter{
								Source: ParameterSourceContext,
								Name:   fieldName.Name,
								Type:   paramTypeIdent,
							})
							continue
						}

						kind := ParameterSourceQuery

						if !IsPrimitive(paramTypeIdent) {
							paramType, err := r.resolver.ResolveType(trctx, paramTypeIdent)
							if err != nil {
								panic(err)
							}

							if paramType == nil {
								panic("type " + paramTypeIdent + " not found")
							}

							switch paramType.Type.(type) {
							case *ast.StructType:
								if _, ok := r.Definitions.Types[paramTypeIdent]; !ok {
									ts := r.ParseType(paramType)
									r.Definitions.Types[paramTypeIdent] = ts
								}
								kind = ParameterSourceBody
							case *ast.InterfaceType:
								if _, ok := r.Definitions.Responders[paramTypeIdent]; !ok {
									resp := r.ParseResponder(trctx, paramType)
									r.Definitions.Responders[paramTypeIdent] = resp
								}
								kind = ParameterSourceResponder
							}
						}

						params = append(params, Parameter{
							Source: kind,
							Name:   fieldName.Name,
							Type:   paramTypeIdent,
						})
					}
				}
			}

			// path params
			for _, pathParam := range pathParamRegex.FindAllStringSubmatch(pathPattern, -1) {
				if len(pathParam) < 2 {
					continue
				}

				paramIdx := slices.IndexFunc(params, func(p Parameter) bool {
					return p.Name == pathParam[1]
				})

				if paramIdx != -1 {
					params[paramIdx].Source = ParameterSourcePath
				}

			}
			// end path params

			// query, header, body params
			if paramsAnnotations, ok := annotations["@Param"]; ok {
				for _, annotation := range paramsAnnotations {
					parts := strings.SplitN(annotation, " ", 3)
					if len(parts) < 2 {
						r.logger.Warn("incorrect annotation", "annotation", "@Param", "value", annotation)
						continue
					}

					name := parts[0]
					source := parts[1]

					paramIdx := slices.IndexFunc(params, func(p Parameter) bool {
						return p.Name == name
					})

					if paramIdx < 0 {
						r.logger.Error("unknown param annotation", "name", name)
						os.Exit(1)
					}

					params[paramIdx].Source = ParameterSource(source)
					if len(parts) == 3 {
						params[paramIdx].Metadata = parts[2]
					}
				}
			}
			// end query, header, body params

			// FIXME: or use full name with package ???
			// TODO: summary, details and tags annotation
			// definitions.Controllers[node.Name.Name] =

			controllerName := ResolveResourceControllerName(node)
			r.Definitions.Controllers[controllerName].Resources[node.Name.Name] = Resource{
				Package: packagePath,
				File:    fileName,

				Name:   node.Name.Name,
				Method: method,
				Path:   pathPattern,
				Params: params,
			}
		}

		return false
	})

}

func ParseAnnotations(comments *ast.CommentGroup) map[string][]string {
	annotations := make(map[string][]string, 0)

	if comments == nil {
		return annotations
	}

	for _, l := range comments.List {
		line := strings.TrimSpace(strings.TrimPrefix(l.Text, "//"))

		if strings.HasPrefix(line, "@") {
			parts := strings.SplitN(line, " ", 2)
			if annotations[parts[0]] == nil {
				annotations[parts[0]] = make([]string, 0)
			}

			if len(parts) > 1 {
				annotations[parts[0]] = append(annotations[parts[0]], parts[1])
			}
		}
	}
	return annotations
}

func ResolveResourceControllerName(node *ast.FuncDecl) string {
	switch t := node.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		return t.X.(*ast.Ident).Name
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func (r *RestCompilerAnalyzer) ParseResponder(trctx TypeResolvingContext, resolvedType *ResolvedType) Responder {
	// TODO: implement me
	// TODO: analyze methods and inputs

	responses := make([]Response, 0)

	i := resolvedType.Type.(*ast.InterfaceType)
	for _, m := range i.Methods.List {

		mf := m.Type.(*ast.FuncType)
		params := make([]Parameter, 0)

		for _, mfp := range mf.Params.List {
			for _, mfpn := range mfp.Names {
				fullTypeName := r.resolver.ResolveIdentifierExpr(trctx, mfp.Type)
				rt, err := r.resolver.ResolveType(trctx, fullTypeName)
				if err != nil {
					panic("cannot resolve responder type")
				}

				if rt == nil {
					panic("resolving type for responder not found: " + fullTypeName)
				}

				// TODO: add to types if not exists

				if _, ok := r.Definitions.Types[fullTypeName]; !ok {
					ts := r.ParseType(rt)
					r.Definitions.Types[fullTypeName] = ts
				}

				params = append(params, Parameter{
					Type: fullTypeName,
					Name: mfpn.Name,
				})
			}
		}

		responses = append(responses, Response{
			Name:        m.Names[0].Name,
			Annotations: ParseAnnotations(m.Doc),
			Params:      params,
		})
	}
	return Responder{
		Name:      resolvedType.Name,
		Responses: responses,
	}
}

func (r *RestCompilerAnalyzer) ParseType(resolvedType *ResolvedType) TypeSchema {
	// TODO: implement me
	return TypeSchema{
		Name: resolvedType.Name,
	}
}
