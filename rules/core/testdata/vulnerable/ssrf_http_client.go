//go:build ignore

package vulnerable

// Lines below a "ruleid:" comment must be reported by that rule; lines below
// an "ok:" comment must not be. Reading a query parameter is not a request.

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProxyHandler(client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ok: qsdev.core.go.ssrf-http-client
		target := r.URL.Query().Get("url")

		// ruleid: qsdev.core.go.ssrf-http-client
		resp, err := http.Get(target)
		if err == nil {
			resp.Body.Close()
		}

		// ruleid: qsdev.core.go.ssrf-http-client
		resp, err = client.Get(target)
		if err == nil {
			resp.Body.Close()
		}
	}
}

func GinProxy(c *gin.Context) {
	// ok: qsdev.core.go.ssrf-http-client
	target := c.Query("url")

	// ruleid: qsdev.core.go.ssrf-http-client
	resp, err := http.Get(target)
	if err == nil {
		resp.Body.Close()
	}
}
