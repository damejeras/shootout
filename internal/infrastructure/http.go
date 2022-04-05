package infrastructure

import "net/http"

func SingleEndpointHTTPServer(port, path string, handlerFunc http.HandlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(path, handlerFunc)

	return &http.Server{
		Addr:    port,
		Handler: mux,
	}
}
