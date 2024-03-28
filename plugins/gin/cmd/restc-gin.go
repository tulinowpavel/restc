package main

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/tulinowpavel/restc"
)

func main() {
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var definitions restc.Definitions
	if err := json.Unmarshal(payload, &definitions); err != nil {
		panic(err)
	}

	sb := strings.Builder{}

	sb.WriteString("// Code generated with RESTc compiler's gin plugin DO NOT EDIT.\n\n")
	sb.WriteString("package server\n\n")

	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/gin-gonic/gin\"\n\n")
	for _, im := range definitions.Imports {
		sb.WriteString("\t")
		sb.WriteString(im)
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	sb.WriteString("\n\n")

	for name, controller := range definitions.Controllers {
		sb.WriteString("func GinRegister")
		sb.WriteString(name)
		sb.WriteString("(r *gin.Engine, c *")
		sb.WriteString(controller.Alias)
		sb.WriteString(") {\n\n")
		for name, resource := range controller.Resources {
			sb.WriteString("\tr.Handle(\"")
			sb.WriteString(resource.Method)
			sb.WriteString("\", \"")
			sb.WriteString(NormalizePath(controller.Base + resource.Path))
			sb.WriteString("\", func(ctx *gin.Context) {\n")
			for _, param := range resource.Params {
				switch param.Source {
				case restc.ParameterSourceHeader:
					metadata := strings.Split(param.Metadata, " ")

					switch param.Type {
					case "string":
						sb.WriteString("\t\t")
						sb.WriteString(param.Name)
						sb.WriteString(" := ctx.GetHeader(\"")
						sb.WriteString(metadata[0])
						sb.WriteString("\")\n")
					default:
						panic("header param type is not supported")
					}

				case restc.ParameterSourcePath:
					switch param.Type {
					case "string":
						sb.WriteString("\t\t")
						sb.WriteString(param.Name)
						sb.WriteString(" := ctx.Param(\"")
						sb.WriteString(param.Name)
						sb.WriteString("\")\n")
					default:
						panic("path param type is not supported")
					}

				case restc.ParameterSourceQuery:
					switch param.Type {
					case "string":
						sb.WriteString("\t\t")
						sb.WriteString(param.Name)
						sb.WriteString(" := ctx.Query(\"")
						sb.WriteString(param.Name)
						sb.WriteString("\")\n")
					default:
						panic("query param type is not supported")

					}

					// TODO: type is slice => GetQuery

				case restc.ParameterSourceBody:
					sb.WriteString("\t\tvar ")
					sb.WriteString(param.Name)
					sb.WriteString(" ")
					sb.WriteString(NormalizeTypeIdentifier(param.Type))
					sb.WriteString("\n")
					sb.WriteString("\t\tif err := ctx.BindJSON(&")
					sb.WriteString(param.Name)
					sb.WriteString("); err != nil {\n")
					sb.WriteString("\t\t\treturn\n")
					sb.WriteString("\t\t}\n")
				}
			}

			sb.WriteString("\n")
			for _, param := range resource.Params {
				if param.Source == restc.ParameterSourceResponder {
					sb.WriteString("\t\t")
					sb.WriteString(param.Name)
					sb.WriteString(" := ")
					sb.WriteString("&gin")
					sb.WriteString(definitions.Responders[param.Type].Name)
					sb.WriteString("{ctx: ctx}\n")
				}
			}
			sb.WriteString("\n")

			sb.WriteString("\t\tif err := c.")
			sb.WriteString(name)
			sb.WriteString("(")
			for idx, param := range resource.Params {
				sb.WriteString(param.Name)
				if idx < len(resource.Params)-1 {
					sb.WriteString(", ")
				}
			}
			sb.WriteString("); err != nil {\n")
			sb.WriteString("\t\t\tctx.Error(err)\n\t\t\tctx.Abort()\n\t\t\treturn\n")
			sb.WriteString("\t\t}\n")
			sb.WriteString("\t})")
			sb.WriteString("\n\n")
		}
		sb.WriteString("}\n\n")
	}

	for _, responder := range definitions.Responders {
		sb.WriteString("type gin")
		sb.WriteString(responder.Name)
		sb.WriteString(" struct {\n")
		sb.WriteString("\tctx *gin.Context\n")
		sb.WriteString("}\n\n")

		for _, response := range responder.Responses {
			sb.WriteString("func (r *gin")
			sb.WriteString(responder.Name)
			sb.WriteString(") ")
			sb.WriteString(response.Name)
			sb.WriteString("(")
			for idx, param := range response.Params {
				sb.WriteString(param.Name)
				sb.WriteString(" ")
				sb.WriteString(NormalizeTypeIdentifier(param.Type))
				if idx < len(response.Params)-1 {
					sb.WriteString(", ")
				}
			}
			sb.WriteString(") {\n")

			status := "200"
			if statusAnnotations, ok := response.Annotations["@Status"]; ok {
				status = statusAnnotations[0]
			}

			if len(response.Params) > 0 {
				sb.WriteString("\tr.ctx.JSON(")
				sb.WriteString(status)
				sb.WriteString(", ")
				sb.WriteString(response.Params[0].Name)
				sb.WriteString(")")
			} else {
				sb.WriteString("\tr.ctx.Status(")
				sb.WriteString(status)
				sb.WriteString(")")
			}

			sb.WriteString("\n}\n\n")
		}
	}

	os.WriteFile(os.ExpandEnv("${RESTC_OUTPUT}"), []byte(sb.String()), 0666)
}

func NormalizePath(resourcePath string) string {
	r := regexp.MustCompile(`\{.+?\}`)
	nameRegex := regexp.MustCompile(`\{([a-zA-Z0-9]+)\}`)
	return r.ReplaceAllStringFunc(resourcePath, func(s string) string {
		matches := nameRegex.FindStringSubmatch(s)
		return ":" + matches[1]
	})
}

var normalizeTypeIdentifierRegex = regexp.MustCompile(`[\/\.\-]+`)

func NormalizeTypeIdentifier(name string) string {
	parts := strings.Split(name, " ")
	if len(parts) == 2 {
		return strings.ToLower(normalizeTypeIdentifierRegex.ReplaceAllString(parts[0], "_")) + "." + parts[1]
	} else {
		return name
	}
}

func ConvertString(name, getter, asType string) string {
	switch asType {
	case "int":
	case "int8":
	case "int16":
	case "int32":
	case "int64":
	case "uint":
	case "uint8":
	case "uint16":
	case "uint32":
	case "uint64":
	case "uintptr":
	case "bool":
	case "rune":
	case "byte":
	}

	return name + " := " + getter
}
