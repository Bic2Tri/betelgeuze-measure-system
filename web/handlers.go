package web

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "betelgeuze-measure-system-main/devices"
    "betelgeuze-measure-system-main/types"
    "betelgeuze-measure-system-main/logging"
    
    "github.com/atotto/clipboard"
)

func statusHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(state.Status)
}

func reconnectHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Закрываем существующие соединения
    if state.Arduino != nil && state.Arduino.Port != nil {
        state.Arduino.Port.Close()
    }
    if state.Scale != nil && state.Scale.Connection != nil {
        state.Scale.Connection.Close()
    }

    // Пытаемся переподключиться
    var err error
    state.Arduino, err = devices.ConnectToArduino()
    if err != nil {
        state.Status.ArduinoConnected = false
        state.Status.ArduinoPort = "Не найден"
    } else {
        state.Status.ArduinoConnected = true
        state.Status.ArduinoPort = state.Arduino.PortName
    }

    state.Scale, err = devices.ConnectToScale()
    if err != nil {
        state.Status.ScaleConnected = false
        state.Status.ScalePort = "Не найден"
    } else {
        state.Status.ScaleConnected = true
        state.Status.ScalePort = state.Scale.PortName
    }

    response := fmt.Sprintf("Arduino: %s (%s), Весы: %s (%s)", 
        boolToString(state.Status.ArduinoConnected), state.Status.ArduinoPort,
        boolToString(state.Status.ScaleConnected), state.Status.ScalePort)
    
    w.Write([]byte(response))
}

func arduinoCommandHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if !state.Status.ArduinoConnected {
        http.Error(w, "Arduino не подключен", http.StatusServiceUnavailable)
        return
    }

    var req struct {
        Command string `json:"command"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    response := devices.ExecuteArduinoCommand(state.Arduino, req.Command)
    w.Write([]byte(response))
}

func scaleReadHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if !state.Status.ScaleConnected {
        http.Error(w, "Весы не подключены", http.StatusServiceUnavailable)
        return
    }

    weight, err := devices.ReadWeight(state.Scale)
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка чтения веса: %v", err), http.StatusInternalServerError)
        return
    }

    state.Status.LastWeight = weight
    response := fmt.Sprintf("%.1f г", weight)
    w.Write([]byte(response))
}

func combinedMeasureHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if !state.Status.ArduinoConnected {
        http.Error(w, "Arduino не подключен", http.StatusServiceUnavailable)
        return
    }

    if !state.Status.ScaleConnected {
        http.Error(w, "Весы не подключены", http.StatusServiceUnavailable)
        return
    }

    // Читаем вес
    weight, err := devices.ReadWeight(state.Scale)
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка чтения веса: %v", err), http.StatusInternalServerError)
        return
    }

    // Получаем размеры
    devices.SendCommandToArduino(state.Arduino, 0x89) // CMD_GET_DIMENSIONS
    time.Sleep(100 * time.Millisecond) // Небольшая задержка для обработки
    length, width, height := devices.GetDimensionsFromArduino(state.Arduino)

    // Формируем результат в формате "вес:высота:ширина:длина"
    result := fmt.Sprintf("%.0f:%d:%d:%d", weight, height, width, length)
    
    // Обновляем статус
    state.Status.LastWeight = weight
    state.Status.LastDimensions = result

    // Копируем в буфер обмена
    err = clipboard.WriteAll(result)
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка копирования в буфер: %v", err), http.StatusInternalServerError)
        return
    }

    // Возвращаем результат
    response := fmt.Sprintf("Измерение завершено: %s (скопировано в буфер)", result)
    w.Write([]byte(response))
}

func logsStreamHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    client := make(chan types.LogMessage, 100)
    logging.AddLogClient(client)
    defer logging.RemoveLogClient(client)

    for {
        select {
        case msg := <-client:
            data, _ := json.Marshal(msg)
            fmt.Fprintf(w, "data: %s\n\n", data)
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
        case <-r.Context().Done():
            return
        }
    }
}

func boolToString(b bool) string {
    if b {
        return "Подключен"
    }
    return "Отключен"
}