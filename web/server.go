package web

import (
    "fmt"
    "log"
    "net/http"
    
    "betelgeuze-measure-system-main/config"
    "betelgeuze-measure-system-main/types"
)

func StartServer(state *types.AppState) {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        indexHandler(w, r, state)
    })
    http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
        statusHandler(w, r, state)
    })
    http.HandleFunc("/reconnect", func(w http.ResponseWriter, r *http.Request) {
        reconnectHandler(w, r, state)
    })
    http.HandleFunc("/arduino/command", func(w http.ResponseWriter, r *http.Request) {
        arduinoCommandHandler(w, r, state)
    })
    http.HandleFunc("/scale/read", func(w http.ResponseWriter, r *http.Request) {
        scaleReadHandler(w, r, state)
    })
    http.HandleFunc("/measure/combined", func(w http.ResponseWriter, r *http.Request) {
        combinedMeasureHandler(w, r, state)
    })
    http.HandleFunc("/logs/stream", func(w http.ResponseWriter, r *http.Request) {
        logsStreamHandler(w, r, state)
    })

    fmt.Println("Веб-сервер запущен на http://localhost" + config.SERVER_PORT)
    log.Fatal(http.ListenAndServe(config.SERVER_PORT, nil))
}