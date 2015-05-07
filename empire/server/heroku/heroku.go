package heroku

import (
	"encoding/json"
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server/authorization"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// New creates the API routes and returns a new http.Handler to serve them.
func New(e *empire.Empire, auth authorization.Authorizer) httpx.Handler {
	r := httpx.NewRouter()

	// Apps
	r.Handle("GET", "/apps", Authenticate(e, &GetApps{e}))                 // hk apps
	r.Handle("DELETE", "/apps/{app}", Authenticate(e, &DeleteApp{e}))      // hk destroy
	r.Handle("POST", "/apps", Authenticate(e, &PostApps{e}))               // hk create
	r.Handle("POST", "/organizations/apps", Authenticate(e, &PostApps{e})) // hk create

	// Domains
	r.Handle("GET", "/apps/{app}/domains", Authenticate(e, &GetDomains{e}))                 // hk domains
	r.Handle("POST", "/apps/{app}/domains", Authenticate(e, &PostDomains{e}))               // hk domain-add
	r.Handle("DELETE", "/apps/{app}/domains/{hostname}", Authenticate(e, &DeleteDomain{e})) // hk domain-remove

	// Deploys
	r.Handle("POST", "/deploys", Authenticate(e, &PostDeploys{e})) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", Authenticate(e, &GetReleases{e}))          // hk releases
	r.Handle("GET", "/apps/{app}/releases/{version}", Authenticate(e, &GetRelease{e})) // hk release-info
	r.Handle("POST", "/apps/{app}/releases", Authenticate(e, &PostReleases{e}))        // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", Authenticate(e, &GetConfigs{e}))     // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", Authenticate(e, &PatchConfigs{e})) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", Authenticate(e, &GetProcesses{e}))                     // hk dynos
	r.Handle("POST", "/apps/{app}/dynos", Authenticate(e, &PostProcess{e}))                     // hk run
	r.Handle("DELETE", "/apps/{app}/dynos", Authenticate(e, &DeleteProcesses{e}))               // hk restart
	r.Handle("DELETE", "/apps/{app}/dynos/{ptype}.{pid}", Authenticate(e, &DeleteProcesses{e})) // hk restart web.1
	r.Handle("DELETE", "/apps/{app}/dynos/{ptype}", Authenticate(e, &DeleteProcesses{e}))       // hk restart web

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", Authenticate(e, &PatchFormation{e})) // hk scale

	// OAuth
	r.Handle("POST", "/oauth/authorizations", &PostAuthorizations{e, auth})

	// SSL
	r.Handle("GET", "/apps/{app}/ssl-endpoints", &GetSSLEndpoints{e})             // hk ssl
	r.Handle("POST", "/apps/{app}/ssl-endpoints", &PostSSLEndpoints{e})           // hk ssl-cert-add
	r.Handle("PATCH", "/apps/{app}/ssl-endpoints/{cert}", &PatchSSLEndpoint{e})   // hk ssl-cert-add, hk ssl-cert-rollback
	r.Handle("DELETE", "/apps/{app}/ssl-endpoints/{cert}", &DeleteSSLEndpoint{e}) // hk ssl-destroy

	errorHandler := func(err error, w http.ResponseWriter, r *http.Request) {
		Error(w, err, http.StatusInternalServerError)
	}

	return middleware.HandleError(r, errorHandler)
}

// Encode json encodes v into w.
func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

// Decode json decodes the request body into v.
func Decode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Stream encodes and flushes data to the client.
func Stream(w http.ResponseWriter, v interface{}) error {
	if err := Encode(w, v); err != nil {
		return err
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// Error is used to respond with errors in the heroku error format, which is
// specified at
// https://devcenter.heroku.com/articles/platform-api-reference#errors
//
// If an ErrorResource is provided as the error, and it provides a non-zero
// status, that will be used as the response status code.
func Error(w http.ResponseWriter, err error, status int) error {
	res := newError(err)

	// If the ErrorResource provides and exit status, we'll use that
	// instead.
	if res.Status != 0 {
		status = res.Status
	}

	w.WriteHeader(status)
	return Encode(w, res)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return Encode(w, nil)
}
