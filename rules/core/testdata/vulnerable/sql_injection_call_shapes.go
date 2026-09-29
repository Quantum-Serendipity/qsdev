//go:build ignore

package vulnerable

// Call shapes the taint sinks must cover beyond the single-argument form:
// bind arguments after a tainted query, the *Context variants, QueryRow*, and
// Prepare*. Lines below a "ruleid:" comment must be reported by that rule;
// lines below an "ok:" comment must not be.

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
)

func TenantSearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		name := r.URL.Query().Get("name")
		tenantID := 42

		query := fmt.Sprintf("SELECT id FROM users WHERE name = '%s' AND tenant = $1", name)
		// ruleid: qsdev.core.go.sqli-database-sql
		rows, _ := db.Query(query, tenantID)
		defer rows.Close()

		// ruleid: qsdev.core.go.sqli-database-sql
		rows2, _ := db.QueryContext(ctx, fmt.Sprintf("SELECT id FROM users WHERE name = '%s'", name), tenantID)
		defer rows2.Close()

		// ruleid: qsdev.core.go.sqli-database-sql
		row := db.QueryRowContext(ctx, "SELECT id FROM users WHERE name = '"+name+"'")
		_ = row

		// ruleid: qsdev.core.go.sqli-database-sql
		row2 := db.QueryRow("SELECT id FROM users WHERE name = '"+name+"' AND tenant = $1", tenantID)
		_ = row2

		// ruleid: qsdev.core.go.sqli-database-sql
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE name = '"+name+"'")

		// ruleid: qsdev.core.go.sqli-database-sql
		stmt, _ := db.PrepareContext(ctx, "SELECT id FROM users WHERE name = '"+name+"'")
		_ = stmt

		// ok: qsdev.core.go.sqli-database-sql
		safeRows, _ := db.QueryContext(ctx, "SELECT id FROM users WHERE name = $1 AND tenant = $2", name, tenantID)
		defer safeRows.Close()

		// ok: qsdev.core.go.sqli-database-sql
		_, _ = db.Exec("UPDATE users SET seen = true WHERE name = $1", name)
	}
}

func ShellHandler(w http.ResponseWriter, r *http.Request) {
	// ruleid: qsdev.core.go.cmdi-exec-command
	out, _ := exec.CommandContext(r.Context(), "sh", "-c", r.FormValue("x")).CombinedOutput()
	_, _ = w.Write(out)

	// ruleid: qsdev.core.go.cmdi-exec-command
	out2, _ := exec.Command("bash", "-c", "grep "+r.FormValue("q"), "extra-arg").CombinedOutput()
	_, _ = w.Write(out2)

	// ok: qsdev.core.go.cmdi-exec-command
	out3, _ := exec.CommandContext(r.Context(), "grep", "--", r.FormValue("q"), "file.txt").CombinedOutput()
	_, _ = w.Write(out3)
}
