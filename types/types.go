package types

import (
    "io"
    "sync"
    
    arduinoSerial "go.bug.st/serial"
)

type ArduinoPort struct {
    Port     arduinoSerial.Port
    PortName string
}

type ScalePort struct {
    Connection io.ReadWriteCloser
    PortName   string
}

type DeviceStatus struct {
    ArduinoConnected bool    `json:"arduino_connected"`
    ArduinoPort      string  `json:"arduino_port"`
    ScaleConnected   bool    `json:"scale_connected"`
    ScalePort        string  `json:"scale_port"`
    LastWeight       float64 `json:"last_weight"`
    LastDimensions   string  `json:"last_dimensions"`
}

type LogMessage struct {
    Time    string `json:"time"`
    Message string `json:"message"`
    Type    string `json:"type"` // "arduino", "scale", "system"
}

type AppState struct {
    Arduino    *ArduinoPort
    Scale      *ScalePort
    Status     DeviceStatus
    LogClients map[chan LogMessage]bool
    LogMutex   sync.RWMutex
}