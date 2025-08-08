package logging

import (
    "sync"
    "time"
    
    "betelgeuze-measure-system-main/types"
)

var (
    logClients = make(map[chan types.LogMessage]bool)
    logMutex   = sync.RWMutex{}
)

func Init() {
    // Инициализация логирования
    BroadcastLog("Система логирования инициализирована", "system")
}

func AddLogClient(client chan types.LogMessage) {
    logMutex.Lock()
    defer logMutex.Unlock()
    logClients[client] = true
}

func RemoveLogClient(client chan types.LogMessage) {
    logMutex.Lock()
    defer logMutex.Unlock()
    delete(logClients, client)
    close(client)
}

func BroadcastLog(message, logType string) {
    logMsg := types.LogMessage{
        Time:    time.Now().Format("15:04:05"),
        Message: message,
        Type:    logType,
    }
    
    logMutex.RLock()
    defer logMutex.RUnlock()
    
    for client := range logClients {
        select {
        case client <- logMsg:
        default:
            // Клиент не готов принять сообщение, пропускаем
        }
    }
}