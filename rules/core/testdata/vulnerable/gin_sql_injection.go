//go:build ignore

package vulnerable

// Lines below a "ruleid:" comment must be reported by that rule; lines below
// an "ok:" comment must not be.

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine, db *sql.DB) {
	r.GET("/users", func(c *gin.Context) {
		// ok: qsdev.core.go.sqli-gin-handler
		name := c.Query("name")

		// ruleid: qsdev.core.go.sqli-gin-handler
		rows, err := db.QueryContext(c, "SELECT id FROM users WHERE name = '"+name+"'", 1)
		if err != nil {
			c.String(http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		// ok: qsdev.core.go.sqli-gin-handler
		safe, _ := db.Query("SELECT id FROM users WHERE name = $1", name)
		defer safe.Close()
	})
}
