package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

const swaggerPage = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Kanban Lab API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="/docs/init.js"></script>
</body>
</html>`

const swaggerInit = `SwaggerUIBundle({ url: '/docs/openapi.json', dom_id: '#ui' })`

const docsCSP = "default-src 'none'; script-src 'self' https://unpkg.com; style-src https://unpkg.com; " +
	"img-src 'self' data: https://unpkg.com; connect-src 'self'; frame-ancestors 'none'"

func docsPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Security-Policy", docsCSP)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerPage))
}

func docsInit(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write([]byte(swaggerInit))
}

func docsSpec(w http.ResponseWriter, _ *http.Request) {
	spec, err := api.GetSpec()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spec)
}
