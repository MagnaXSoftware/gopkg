package gopkg

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"html/template"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

//DefaultTemplate is the default HTML template used as a response.
const DefaultTemplate = `<html>
<head>
<meta name="go-import" content="{{.Host}}{{.Path}} {{.Vcs}} {{.URI}}">
</head>
<body>
go get {{.Host}}{{.Path}}
</body>
</html>
`

func init() {
	caddy.RegisterModule(Module{})
	httpcaddyfile.RegisterDirective("gopkg", parseCaddyFile)
}

//Module represents the GoPkg Caddy module.
type Module struct {
	Path string `json:"path"`
	Vcs  string `json:"vcs,omitempty"`
	URI  string `json:"uri"`

	// Template is the template used when returning a response (instead of redirecting).
	Template *template.Template
}

func (m Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.gopkg",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

// parseCaddyFile parses the gopkg directive in a caddyfile.
//
// It also automatically sets up a path matcher so that each instance is separate.
func parseCaddyFile(h httpcaddyfile.Helper) ([]httpcaddyfile.ConfigValue, error) {
	if !h.Next() {
		return nil, h.ArgErr()
	}

	var m = new(Module)
	err := m.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}

	matcher := caddy.ModuleMap{
		"path": h.JSON(m.Path),
	}

	return h.NewRoute(matcher, m), nil

}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
//     gopkg <path> [<vcs>] <uri>
//
func (m *Module) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&m.Path) {
			return d.ArgErr()
		}
		args := d.RemainingArgs()
		switch len(args) {
		case 2:
			d.Args(&m.Vcs)
			fallthrough
		case 1:
			// no vcs
			d.Args(&m.URI)
		default:
			return d.ArgErr()
		}
	}

	return nil
}

func (m *Module) Provision(ctx caddy.Context) error {
	if m.Vcs == "" {
		m.Vcs = "git"
	}

	if m.Template == nil {
		tpl, err := template.New("Package").Parse(DefaultTemplate)
		if err != nil {
			return fmt.Errorf("parsing default gopkg template: %v", err)
		}
		m.Template = tpl
	}

	return nil
}

func (m Module) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// If go-get is not present, it's most likely a browser request. So let's redirect.
	if r.FormValue("go-get") != "1" {
		http.Redirect(w, r, m.URI, http.StatusTemporaryRedirect)
		return nil
	}

	err := m.Template.Execute(w, struct {
		Host string
		Path string
		Vcs  string
		URI  string
	}{r.Host, m.Path, m.Vcs, m.URI})

	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	w.Header().Set("Content-Type", "text/html")
	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Module)(nil)
	_ caddyhttp.MiddlewareHandler = (*Module)(nil)
	_ caddyfile.Unmarshaler       = (*Module)(nil)
)
