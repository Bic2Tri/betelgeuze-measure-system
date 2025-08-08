package utils

import (
    "fmt"
    "strings"
)

func BoolToString(b bool) string {
    if b {
        return "Подключен"
    }
    return "Отключен"
}

func FormatDataForLog(data []byte) string {
    if len(data) == 0 {
        return "нет данных"
    }
    
    // Проверяем, содержит ли данные структурированные блоки Arduino
    hasArduinoBlocks := false
    for i := 0; i < len(data)-3; i++ {
        if data[i] == 0x2D && data[i+3] == 0x7B {
            hasArduinoBlocks = true
            break
        }
    }
    
    // Пробуем преобразовать в строку
    printableStr := ""
    hasText := false
    binaryPart := make([]byte, 0)
    
    for i, b := range data {
        if b >= 32 && b <= 126 { // Печатаемые ASCII символы
            printableStr += string(b)
            hasText = true
        } else if b == 10 { // Перевод строки
            printableStr += "\\n"
            hasText = true
        } else if b == 13 { // Возврат каретки
            printableStr += "\\r"
            hasText = true
        } else if b == 9 { // Табуляция
            printableStr += "\\t"
            hasText = true
        } else {
            // Если это часть структурированных данных Arduino, сохраняем для декодирования
            if hasArduinoBlocks && (b == 0x2D || b == 0x7B || (i > 0 && data[i-1] == 0x2D) || (i < len(data)-1 && data[i+1] == 0x7B)) {
                binaryPart = append(binaryPart, b)
                printableStr += fmt.Sprintf("\\x%02X", b)
            } else {
                printableStr += fmt.Sprintf("\\x%02X", b)
            }
        }
    }
    
    result := ""
    
    // Если есть читаемый текст, показываем его
    if hasText {
        result = fmt.Sprintf("\"%s\"", printableStr)
    }
    
    // Если есть структурированные данные Arduino, декодируем их
    if hasArduinoBlocks && len(binaryPart) >= 4 {
        // Ищем все структурированные блоки в исходных данных
        var structuredBlocks []byte
        for i := 0; i < len(data)-3; i++ {
            if data[i] == 0x2D && data[i+3] == 0x7B {
                // Нашли блок, добавляем его
                if i+4 <= len(data) {
                    structuredBlocks = append(structuredBlocks, data[i:i+4]...)
                }
            }
        }
        
        if len(structuredBlocks) >= 4 {
            decodedData := DecodeArduinoSensorData(structuredBlocks)
            if result != "" {
                result += " + " + decodedData
            } else {
                result = decodedData
            }
        }
    }
    
    // Если ничего не декодировалось, показываем в hex формате
    if result == "" {
        hexStr := make([]string, len(data))
        for i, b := range data {
            hexStr[i] = fmt.Sprintf("0x%02X", b)
        }
        result = fmt.Sprintf("[%s]", strings.Join(hexStr, " "))
    }
    
    return fmt.Sprintf("%s (%d байт)", result, len(data))
}

func DecodeArduinoSensorData(data []byte) string {
    if len(data) < 4 {
        return fmt.Sprintf("Данные слишком короткие: %d байт", len(data))
    }
    
    var result []string
    
    // Проходим по данным блоками по 4 байта
    for i := 0; i < len(data)-3; i += 4 {
        block := data[i:i+4]
        
        // Проверяем структуру блока: должно быть 0x2D (45) в начале и 0x7B (123) в конце
        if block[0] == 0x2D && block[3] == 0x7B {
            sensorID := block[1]
            value := int(block[2])
            
            // Декодируем ID сенсора в понятное название
            sensorName := decodeSensorID(sensorID)
            
            result = append(result, fmt.Sprintf("%s=%d", sensorName, value))
        } else {
            // Если структура не совпадает, показываем как есть
            hexStr := make([]string, 4)
            for j := 0; j < 4; j++ {
                hexStr[j] = fmt.Sprintf("0x%02X", block[j])
            }
            result = append(result, fmt.Sprintf("[%s]", strings.Join(hexStr, " ")))
        }
    }
    
    if len(result) > 0 {
        return fmt.Sprintf("Данные сенсоров: {%s}", strings.Join(result, ", "))
    }
    
    return "Не удалось декодировать данные сенсоров"
}

// Функция для декодирования ID сенсора в понятное название
func decodeSensorID(id byte) string {
    switch id {
    case 0x0B:
        return "WIDTH"  // Ширина
    case 0x16:
        return "HEIGHT" // Высота  
    case 0x21:
        return "LENGTH" // Длина
    case 0xBB:
        return "Right Sensor" // Неизвестный сенсор
    default:
        return fmt.Sprintf("SENSOR_0x%02X", id)
    }
}