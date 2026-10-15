package site

import (
	"net/http"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// A site whose files share its own domain would otherwise run an uploaded page
// with the visitor's session. The sandbox keeps the scripts but takes the origin
// away, and PDFs are spared because browsers refuse to show them sandboxed.
func Sandbox(w http.ResponseWriter, current *db.Site, mimeType string) {
	if current == nil || (current.MediaDomain != "" && !strings.EqualFold(current.MediaDomain, current.Domain)) {
		return
	}
	if strings.HasPrefix(strings.ToLower(mimeType), "application/pdf") {
		return
	}
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-popups allow-modals allow-downloads")
}
