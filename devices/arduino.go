package devices

import (
    "errors"
    "fmt"
    "strings"
    "time"
    "strconv"
    
    "betelgeuze-measure-system-main/config"
    "betelgeuze-measure-system-main/logging"
    "betelgeuze-measure-system-main/types"
    "betelgeuze-measure-system-main/utils"
    
    arduinoSerial "go.bug.st/serial"
    "go.bug.st/serial/enumerator"
)

func ConnectToArduino() (*types.ArduinoPort, error) {
    ports, err := enumerator.GetDetailedPortsList()
    if err != nil {
        return nil, err
    }

    fmt.Println("🔍 Searching for Arduino via PING...")

    for _, port := range ports {
        if !port.IsUSB {
            continue
        }

        fmt.Printf("🔌 Trying port: %s (VID: %s, PID: %s, Product: %s)\n", port.Name, port.VID, port.PID, port.Product)

        mode := &arduinoSerial.Mode{BaudRate: 115200}
        conn, err := arduinoSerial.Open(port.Name, mode)
        if err != nil {
            fmt.Printf("  ❌ Failed to open %s: %v\n", port.Name, err)
            continue
        }

        time.Sleep(2000 * time.Millisecond) // Wait for bootloader to finish

        flush(conn)

        conn.Write([]byte{config.CMD_PING})
        conn.SetReadTimeout(200 * time.Millisecond)

        allData := make([]byte, 0, 512)
        start := time.Now()
        for time.Since(start) < 2*time.Second {
            buf := make([]byte, 64)
            n, _ := conn.Read(buf)
            if n > 0 {
                allData = append(allData, buf[:n]...)
            }
            time.Sleep(10 * time.Millisecond)
        }

        responseStr := string(allData)
        fmt.Printf("  📥 Full response from %s (%d bytes):\n%s\n", port.Name, len(allData), responseStr)

        if strings.Contains(responseStr, "OK") {
            fmt.Printf("  ✅ Arduino detected on port %s\n", port.Name)
            return &types.ArduinoPort{Port: conn, PortName: port.Name}, nil
        }

        conn.Close()
        fmt.Printf("  ⚠️  No OK response found on %s\n", port.Name)
    }

    return nil, errors.New("Arduino not found via PING")
}

func SendCommandToArduino(a *types.ArduinoPort, cmd byte) {
    a.Port.Write([]byte{cmd})
    time.Sleep(200 * time.Millisecond)
}

func GetDimensionsFromArduino(a *types.ArduinoPort) (int, int, int) {
    // Очищаем буфер перед чтением
    flush(a.Port)
    
    // Отправляем команду
    a.Port.Write([]byte{config.CMD_GET_DIMENSIONS})
    logging.BroadcastLog("Отправлена команда GET_DIMENSIONS (0x89)", "arduino")
    
    // Читаем данные в течение 600ms
    a.Port.SetReadTimeout(50 * time.Millisecond)
    allData := make([]byte, 0, 200)
    
    startTime := time.Now()
    for time.Since(startTime) < 600*time.Millisecond {
        buf := make([]byte, 50)
        n, err := a.Port.Read(buf)
        if err == nil && n > 0 {
            allData = append(allData, buf[:n]...)
            
            // Логируем полученные данные в читаемом формате
            dataStr := utils.FormatDataForLog(buf[:n])
            logging.BroadcastLog(fmt.Sprintf("Получены данные: %s", dataStr), "arduino")
        }
        time.Sleep(10 * time.Millisecond)
    }
    
    totalDataStr := utils.FormatDataForLog(allData)
    logging.BroadcastLog(fmt.Sprintf("Всего получено: %s", totalDataStr), "arduino")
    
    if len(allData) < 41 {
        logging.BroadcastLog("Недостаточно данных для парсинга размеров", "arduino")
        return 0, 0, 0
    }
    
    // Ищем начало валидных данных
    validData := findValidDataPattern(allData)
    if len(validData) < 41 {
        logging.BroadcastLog("Не найден валидный паттерн данных", "arduino")
        return 0, 0, 0
    }
    
    length, width, height := parseDimensionsData(validData)
    logging.BroadcastLog(fmt.Sprintf("Распознаны размеры: Длина=%d, Ширина=%d, Высота=%d", length, width, height), "arduino")
    
    return length, width, height
}

func ExecuteArduinoCommand(arduino *types.ArduinoPort, command string) string {
    parts := strings.Split(command, ":")
    cmd := parts[0]
    
    switch cmd {
    case "start":
        SendCommandToArduino(arduino, config.CMD_START)
        return "Команда START отправлена"
    
    case "ping":
        flush(arduino.Port)
        SendCommandToArduino(arduino, config.CMD_PING)
        logging.BroadcastLog("Отправлена команда PING (0x77)", "arduino")
        
        // Читаем данные в течение ~500ms
        arduino.Port.SetReadTimeout(50 * time.Millisecond)
        allData := make([]byte, 0, 200)
        
        startTime := time.Now()
        for time.Since(startTime) < 700*time.Millisecond {
            buf := make([]byte, 20)
            n, err := arduino.Port.Read(buf)
            if err == nil && n > 0 {
                allData = append(allData, buf[:n]...)
                
                // Логируем полученные данные
                hexStr := make([]string, n)
                decStr := make([]string, n)
                for i := 0; i < n; i++ {
                    hexStr[i] = fmt.Sprintf("0x%02X", buf[i])
                    decStr[i] = fmt.Sprintf("%d", buf[i])
                }
                logging.BroadcastLog(fmt.Sprintf("PING ответ: %d байт HEX:[%s] DEC:[%s] ASCII:%s", 
                    n, strings.Join(hexStr, ","), strings.Join(decStr, ","), string(buf[:n])), "arduino")
            }
            time.Sleep(10 * time.Millisecond)
        }
        
        if len(allData) == 0 {
            logging.BroadcastLog("Нет ответа от Arduino на PING", "arduino")
            return "Нет ответа от Arduino"
        }
        
        // Преобразуем полученные данные в строку
        response := string(allData)
        
        // Ищем "OK" в ответе
        if strings.Contains(response, "OK") {
            logging.BroadcastLog("Arduino ответил корректно: OK", "arduino")
            return "Arduino ответил: OK"
        } else {
            // Если "OK" не найдено, показываем что получили
            logging.BroadcastLog(fmt.Sprintf("Arduino ответил нестандартно: %s", strings.TrimSpace(response)), "arduino")
            return fmt.Sprintf("Arduino ответил: %s (%d байт, 'OK' не найдено)", 
                strings.TrimSpace(response), len(allData))
        }
    
    case "reset_sensors":
        SendCommandToArduino(arduino, config.CMD_RESET_SENSORS)
        return "Сенсоры сброшены"
    
    case "led_on":
        SendCommandToArduino(arduino, config.CMD_LED_ON)
        return "Светодиоды включены"
    
    case "led_off":
        SendCommandToArduino(arduino, config.CMD_LED_OFF)
        return "Светодиоды выключены"
    
    case "get_dimensions":
        SendCommandToArduino(arduino, config.CMD_GET_DIMENSIONS)
        length, width, height := GetDimensionsFromArduino(arduino)
        return fmt.Sprintf("Размеры: Д=%d, Ш=%d, В=%d", length, width, height)
    
    case "set_top_max":
        if len(parts) < 2 {
            return "Не указано значение"
        }
        value, err := strconv.Atoi(parts[1])
        if err != nil || value < 1 || value > 255 {
            return "Неверное значение (1-255)"
        }
        arduino.Port.Write([]byte{config.CMD_SET_TOP_MAX, byte(value)})
        return fmt.Sprintf("Максимальная высота установлена: %d", value)
    
    case "set_width_max":
        if len(parts) < 2 {
            return "Не указано значение"
        }
        value, err := strconv.Atoi(parts[1])
        if err != nil || value < 1 || value > 255 {
            return "Неверное значение (1-255)"
        }
        arduino.Port.Write([]byte{config.CMD_SET_WIDTH_MAX, byte(value)})
        return fmt.Sprintf("Максимальная ширина установлена: %d", value)
    
    case "set_length_max":
        if len(parts) < 2 {
            return "Не указано значение"
        }
        value, err := strconv.Atoi(parts[1])
        if err != nil || value < 1 || value > 255 {
            return "Неверное значение (1-255)"
        }
        arduino.Port.Write([]byte{config.CMD_SET_LENGTH_MAX, byte(value)})
        return fmt.Sprintf("Максимальная длина установлена: %d", value)
    
    default:
        return "Неизвестная команда"
    }
}

// Вспомогательные функции
func flush(port arduinoSerial.Port) {
    port.SetReadTimeout(100 * time.Millisecond)
    buf := make([]byte, 256)
    for {
        n, err := port.Read(buf)
        if err != nil || n == 0 {
            break
        }
    }
    port.SetReadTimeout(0)
}

func findValidDataPattern(data []byte) []byte {
    for i := 0; i <= len(data)-41; i++ {
        if data[i] == 0x2D {
            validBlocks := 0
            for j := 0; j < 10 && i+j*4+3 < len(data); j++ {
                offset := i + j*4
                if offset+3 < len(data) && data[offset] == 0x2D && data[offset+3] == 0x7B {
                    validBlocks++
                } else {
                    break
                }
            }
            
            if validBlocks >= 8 {
                if i+41 <= len(data) {
                    return data[i:i+41]
                }
            }
        }
    }
    return data
}

func parseDimensionsData(buf []byte) (int, int, int) {
    var width_box, height_box, length_box int

    for i := 0; i < 10 && i*4+3 < len(buf); i++ {
        offset := i * 4
        block := buf[offset : offset+4]
        
        if block[0] != 0x2D || block[3] != 0x7B {
            continue
        }

        sensorID := block[1]
        value := int(block[2])

        switch sensorID {
        case 0x0B:
            if i == 7 {
                width_box = value
            }
        case 0x16:
            if i == 8 {
                height_box = value
            }
        case 0x21:
            if i == 9 {
                length_box = value
            }
        }
    }
    
    return length_box, width_box, height_box
}