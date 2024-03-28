package restc

type Definitions struct {
	Imports []string `json:"imports"`

	Types       map[string]TypeSchema `json:"types"`
	Responders  map[string]Responder  `json:"responders"`
	Controllers map[string]Controller `json:"controllers"`
}

func NewDefinitions() Definitions {
	return Definitions{
		Types:       make(map[string]TypeSchema),
		Responders:  make(map[string]Responder),
		Controllers: make(map[string]Controller),
	}
}

type TypeSchema struct {
	Name   string  `json:"name"`
	Alias  string  `json:"alias"`
	Schema *Schema `json:"schema,omitempty"`
}

type Responder struct {
	Name      string     `json:"name"`
	Responses []Response `json:"responses,omitempty"`
}

type Response struct {
	Name        string              `json:"name"`
	Annotations map[string][]string `json:"annotations"`
	Params      []Parameter         `json:"params"`
}

type Parameter struct {
	Source   ParameterSource `json:"kind,omitempty"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Metadata string          `json:"metadata,omitempty"`

	Schema *Schema `json:"schema,omitempty"`
}

type ParameterSource string

const (
	ParameterSourceContext   ParameterSource = "Context"
	ParameterSourceResponder ParameterSource = "Responder"
	ParameterSourceHeader    ParameterSource = "Header"
	ParameterSourcePath      ParameterSource = "Path"
	ParameterSourceQuery     ParameterSource = "Query"
	ParameterSourceBody      ParameterSource = "Body"
)

type Controller struct {
	// FIXME: for what
	Package string `json:"package"`
	File    string `json:"file"`

	Name  string `json:"name"`
	Alias string `json:"alias"`

	Base string `json:"base"`

	Resources map[string]Resource `json:"resources"`
}

type Resource struct {
	Package string `json:"package"`
	File    string `json:"file"`

	Name   string      `json:"name"`
	Method string      `json:"method"`
	Path   string      `json:"path"`
	Params []Parameter `json:"params"`

	Summary string   `json:"summary,omitempty"`
	Details string   `json:"details,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}
