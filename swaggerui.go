package swaggerUI

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"code.vegaprotocol.io/vega/logging"
	"github.com/getkin/kin-openapi/openapi2"
)

//go:embed assets
var swagfs embed.FS

type SwaggerUI struct {
	*http.ServeMux
	specFS   fs.FS
	specPath string
}

func New(log *logging.Logger, name string, specFS fs.FS, specPath string) (*SwaggerUI, error) {
	log = log.Named("swagger-ui")

	s := &SwaggerUI{
		ServeMux: http.NewServeMux(),
		name:     name,
		specFS:   specFS,
		specPath: specPath,
	}

	static, _ := fs.Sub(swagfs, "assets")
	specHandler, err := s.specFile()
	if err != nil {
		return nil, err
	}

	mux.HandleFunc("/swagger_spec", specHandler)
	mux.HandleFunc("/swagger-initializer.js", s.swaggerSetup())
	mux.Handle("/", http.FileServer(http.FS(static)))
	s.mux = mux
	return nil

	return s
}

// specFile just returns the openapi2 spec file, but modified slightly to have the correct
// base path, so that the 'Try It Out' functionality works.
func (s *SwaggerUI) specFile() (http.HandlerFunc, error) {
	var spec openapi2.T

	originalSpec, err := fs.ReadFile(s.specFS, s.specPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read embedded openapi json file '%s' (must be generated before build with 'make proto_docs')", blockExplorerSpecPath)
	}

	if err := json.Unmarshal(originalSpec, &spec); err != nil {
		return nil, fmt.Errorf("un-marshalling OpenAPI spec: %w", err)
	}

	spec.BasePath = s.restConfig.Endpoint

	newSpec, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling modified OpenAPI spec: %w", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write(newSpec)
	}, nil
}

// swaggerSetup returns the bit of javascript that sets up swagger, modified
// so that it points at our OpenAPI file endpoint.
func (s *SwaggerUI) swaggerSetup() http.HandlerFunc {
	template := `
window.onload = function () {
	window.ui = SwaggerUIBundle({
	  url: "%s",
	  name: "%s",
	  dom_id: '#swagger-ui',
	  deepLinking: true,
	  presets: [
		SwaggerUIBundle.presets.apis,
		SwaggerUIStandalonePreset
	  ],
	  plugins: [
		SwaggerUIBundle.plugins.DownloadUrl
	  ],
	  layout: "StandaloneLayout"
	});
  };
`
	js := []byte(fmt.Sprintf(template, s.Endpoint+"/swagger_spec", s.Name))
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write(js)
	}
}
