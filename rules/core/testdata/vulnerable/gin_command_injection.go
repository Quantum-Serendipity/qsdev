//go:build ignore

package vulnerable

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func RegisterDiagRoutes(r *gin.Engine) {
	r.GET("/ping/:host", func(c *gin.Context) {
		host := c.Param("host")

		// VULNERABLE: user input passed to shell via sh -c
		cmd := exec.Command("sh", "-c", fmt.Sprintf("ping -c 1 %s", host))
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.String(http.StatusInternalServerError, "ping failed")
			return
		}
		c.String(http.StatusOK, string(output))
	})

	r.GET("/lookup", func(c *gin.Context) {
		domain := c.Query("domain")

		// VULNERABLE: concatenation into shell command
		cmd := exec.Command("bash", "-c", "nslookup "+domain)
		output, _ := cmd.CombinedOutput()
		c.String(http.StatusOK, string(output))
	})
}
