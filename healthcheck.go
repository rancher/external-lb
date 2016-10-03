package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

var (
	router          = mux.NewRouter()
	healthcheckPort = ":1000"
)

func startHealthcheck() {
	router.HandleFunc("/", healthcheck).Methods("GET", "HEAD").Name("Healthcheck")
	logrus.Info("Healthcheck handler is listening on ", healthcheckPort)
	logrus.Fatal(http.ListenAndServe(healthcheckPort, router))
}

func healthcheck(w http.ResponseWriter, req *http.Request) {
	// 1) test metadata server
	_, err := m.MetadataClient.GetSelfStack()
	if err != nil {
		logrus.Error("Metadata health check failed: %v", err)
		http.Error(w, "Failed to reach metadata server", http.StatusInternalServerError)
	} else {
		// 2) test provider
		if err := provider.HealthCheck(); err != nil {
			logrus.Errorf("Provider health check failed: %v", err)
			http.Error(w, "Failed to reach external provider ", http.StatusInternalServerError)
		} else {
			w.Write([]byte("OK"))
		}
	}
}
