package restc

import "slices"

var primitives []string = []string{
	"bool",
	"string",
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"uintptr",
	"rune",
	"byte",
	"float32",
	"float64",
	"complex64",
	"complex128",
}

func IsPrimitive(typeIdentifier string) bool {
	return slices.Contains(primitives, typeIdentifier)
}
