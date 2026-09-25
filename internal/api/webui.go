package api

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"fapi/web"
)

// webUIRoute answers every path outside /api/: the web UI when this build
// includes it and the webUi feature is on, otherwise a message saying why not.
func webUIRoute(featureOn bool) http.HandlerFunc {
	if !web.Included {
		return func(writer http.ResponseWriter, request *http.Request) {
			writeError(writer, http.StatusNotFound, "This build of fapi doesn't include the web UI; use the fapi or fapi-web edition for it")
		}
	}
	return requireFeature(featureOn, "webUi", webUIHandler())
}

// webUIHandler serves the web UI's files, which are built into the
// executable. Paths that aren't files get index.html, so the UI's own
// client-side routes still work when the page is reloaded.
func webUIHandler() http.HandlerFunc {
	files, built := web.Files()
	if !built {
		return serveNotBuiltPage
	}

	fileServer := http.FileServerFS(files)
	return func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if name == "" {
			fileServer.ServeHTTP(writer, request) // serves index.html
			return
		}

		_, err := fs.Stat(files, name)
		if err != nil {
			http.ServeFileFS(writer, request, files, "index.html")
			return
		}
		fileServer.ServeHTTP(writer, request)
	}
}

// notBuiltPage is shown when fapi was built with `go build` alone, before
// the web UI was built. Everything except the web UI still works.
const notBuiltPage = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>fapi</title>
<style>body { font-family: system-ui, sans-serif; max-width: 40rem; margin: 3rem auto; padding: 0 1rem; line-height: 1.5 }
code, pre { background: #8882; padding: 0.1rem 0.3rem; border-radius: 4px }</style></head>
<body>
<h1>fapi is running, but its web UI wasn't built</h1>
<p>This copy of fapi was built without the web UI. The mock servers and the admin API under <code>/api/</code> work as usual.</p>
<p>To include the web UI, build it and then build fapi again, from the root of the fapi repository:</p>
<pre>cd web
npm install
npm run build
cd ..
go build ./cmd/fapi</pre>
</body>
</html>
`

func serveNotBuiltPage(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := writer.Write([]byte(notBuiltPage))
	if err != nil {
		log.Printf("Admin API: sending the page: %v", err)
	}
}
