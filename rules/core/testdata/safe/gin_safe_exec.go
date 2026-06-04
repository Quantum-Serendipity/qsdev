//go:build ignore

package safe

import (
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func RegisterDiagRoutes(r *gin.Engine) {
	r.GET("/ping/:host", func(c *gin.Context) {
		host := c.Param("host")

		// SAFE: arguments passed as separate elements, no shell
		cmd := exec.Command("ping", "-c", "1", host)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.String(http.StatusInternalServerError, "ping failed")
			return
		}
		c.String(http.StatusOK, string(output))
	})

	r.GET("/readlog", func(c *gin.Context) {
		filename := c.Query("file")

		// SAFE: filepath.Base strips directory traversal, no shell
		safeName := filepath.Base(filename)
		cmd := exec.Command("cat", "/var/log/app/"+safeName)
		output, _ := cmd.CombinedOutput()
		c.String(http.StatusOK, string(output))
	})
}
