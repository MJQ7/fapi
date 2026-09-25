package core

import "net/http"

// Browsers only let a page read a response from another origin (such as a
// frontend on localhost:5173 calling fapi on localhost:3001) when the
// response carries CORS headers. For requests that aren't "simple", the
// browser first asks permission with an OPTIONS "preflight" request.
// fapi answers those itself and allows everything, since it's a test tool.
// None of this applies to the admin API.

// isPreflight reports whether request is a browser's CORS preflight check.
func isPreflight(request *http.Request) bool {
	return request.Method == http.MethodOptions &&
		request.Header.Get("Origin") != "" &&
		request.Header.Get("Access-Control-Request-Method") != ""
}

// writePreflight allows whatever the preflight asked for.
func writePreflight(writer http.ResponseWriter, request *http.Request) {
	headers := writer.Header()
	headers.Set("Access-Control-Allow-Origin", request.Header.Get("Origin"))
	headers.Set("Access-Control-Allow-Methods", request.Header.Get("Access-Control-Request-Method"))

	requestedHeaders := request.Header.Get("Access-Control-Request-Headers")
	if requestedHeaders != "" {
		headers.Set("Access-Control-Allow-Headers", requestedHeaders)
	}

	headers.Set("Access-Control-Allow-Credentials", "true")
	headers.Set("Access-Control-Max-Age", "600")
	headers.Add("Vary", "Origin")
	writer.WriteHeader(http.StatusNoContent)
}

// addCORSHeaders lets a page on origin read the response.
func addCORSHeaders(headers http.Header, origin string) {
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Credentials", "true")
	headers.Set("Access-Control-Expose-Headers", "*")
	headers.Add("Vary", "Origin")
}
