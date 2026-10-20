package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"service-platform/internal/api"
)

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Go Service Platform API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui'
    });
  </script>
</body>
</html>`

func openAPIHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec, err := api.GetSwagger()
		if err != nil {
			logger.Error("loading OpenAPI specification", slog.Any("error", err))
			writeError(logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(spec); err != nil {
			logger.Error("encoding OpenAPI specification", slog.Any("error", err))
		}
	}
}

func swaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUIHTML))
	}
}
