package main

import (
    "fmt"
    "log"
    "runtime"
    "time"
    
    "betelgeuze-measure-system-main/config"
    "betelgeuze-measure-system-main/devices"
    "betelgeuze-measure-system-main/logging"
    "betelgeuze-measure-system-main/types"
    "betelgeuze-measure-system-main/web"
    "betelgeuze-measure-system-main/utils"
    "math"
    
    "github.com/atotto/clipboard"
    "github.com/micmonay/keybd_event"
)

func main() {
    // Инициализация системы логирования
    logging.Init()
    
    // Создание глобального состояния
    appState := &types.AppState{}
    
    // Инициализация Arduino
    fmt.Println("🔌 Поиск Arduino...")
    arduino, err := devices.ConnectToArduino()
    if err != nil {
        log.Println("Arduino error:", err)
        appState.Status.ArduinoConnected = false
        appState.Status.ArduinoPort = "Не найден"
    } else {
        fmt.Printf("✅ Arduino подключен: %s\n", arduino.PortName)
        appState.Arduino = arduino
        appState.Status.ArduinoConnected = true
        appState.Status.ArduinoPort = arduino.PortName
        defer arduino.Port.Close()
    }
    
    // Инициализация весов
    fmt.Println("⚖️ Поиск весов...")
    scale, err := devices.ConnectToScale()
    if err != nil {
        log.Println("Scale error:", err)
        appState.Status.ScaleConnected = false
        appState.Status.ScalePort = "Не найден"
    } else {
        fmt.Printf("✅ Весы подключены: %s\n", scale.PortName)
        appState.Scale = scale
        appState.Status.ScaleConnected = true
        appState.Status.ScalePort = scale.PortName
        defer scale.Connection.Close()
    }
    
    // Запуск веб-сервера
    go web.StartServer(appState)
    
    // Основной цикл работы
    if appState.Status.ScaleConnected {
        mainLoop(appState)
    } else {
        log.Println("Весы не подключены. Используйте веб-интерфейс для управления.")
        select {} // Блокируем выполнение
    }
}

func printStatus(state *types.AppState) {
    fmt.Println("📡 Статус подключения устройств:")
    fmt.Printf("🔌 Arduino: %s (%s)\n", utils.BoolToString(state.Status.ArduinoConnected), state.Status.ArduinoPort)
    fmt.Printf("⚖️ Весы: %s (%s)\n", utils.BoolToString(state.Status.ScaleConnected), state.Status.ScalePort)
}

// Кроссплатформенная функция для симуляции нажатий клавиш
func simulateKeyPress(result string) error {
    // Копируем результат в буфер обмена
    err := clipboard.WriteAll(result)
    if err != nil {
        return fmt.Errorf("ошибка буфера обмена: %v", err)
    }

    // Создаем клавиатурное событие с учетом ОС
    kb, err := keybd_event.NewKeyBonding()
    if err != nil {
        return fmt.Errorf("ошибка создания клавиатурного события: %v", err)
    }

    // Устанавливаем правильный модификатор в зависимости от ОС
    if runtime.GOOS == "darwin" { // macOS
        kb.HasSuper(true)
    } else { // Windows и Linux
        kb.HasCTRL(true)
    }
    
    kb.SetKeys(keybd_event.VK_V)
    
    // Задержка перед нажатием (важно для стабильности)
    time.Sleep(200 * time.Millisecond)
    
    err = kb.Launching()
    if err != nil {
        return fmt.Errorf("ошибка симуляции Ctrl+V: %v", err)
    }

    // Создаем новое событие для Enter
    kbEnter, err := keybd_event.NewKeyBonding()
    if err != nil {
        return fmt.Errorf("ошибка создания события Enter: %v", err)
    }
    
    // Небольшая задержка между нажатиями
    time.Sleep(100 * time.Millisecond)
    
    kbEnter.SetKeys(keybd_event.VK_ENTER)
    err = kbEnter.Launching()
    if err != nil {
        return fmt.Errorf("ошибка симуляции Enter: %v", err)
    }

    return nil
}

func mainLoop(state *types.AppState) {
    var lastWeight float64 = -1 // Инициализируем значением, которое точно не может быть реальным весом
    const weightThreshold = config.WEIGHT_THRESHOLD  // Минимальное изменение веса для запуска измерения (в граммах)
    
    fmt.Printf("🖥️ Система запущена на %s\n", runtime.GOOS)
    
    // Определяем режим работы
    if state.Status.ArduinoConnected && state.Status.ScaleConnected {
        fmt.Println("🔄 Режим работы: Полные измерения (весы + Arduino)")
    } else if state.Status.ScaleConnected {
        fmt.Println("⚖️ Режим работы: Только весы")
    }
    
    for {
        if !state.Status.ScaleConnected {
            time.Sleep(1 * time.Second)
            continue
        }

        // Инициализируем Arduino только если он подключен
        if state.Status.ArduinoConnected {
            devices.SendCommandToArduino(state.Arduino, config.CMD_START)
            time.Sleep(2 * time.Second)
        }

        for {
            weight, err := devices.ReadWeight(state.Scale)
            if err != nil {
                fmt.Println("Ошибка чтения веса:", err)
                time.Sleep(1 * time.Second)
                continue
            }
            
            // Проверяем, что вес больше 0 (есть объект на весах)
            if weight <= 0 {
                time.Sleep(1 * time.Second)
                continue
            }
            
            // Проверяем изменение веса с учетом порога
            if lastWeight > 0 && math.Abs(weight-lastWeight) < weightThreshold {
                time.Sleep(1 * time.Second)
                continue
            }
            
            // Обновляем последний вес
            lastWeight = weight
            state.Status.LastWeight = weight
            
            fmt.Printf("🔄 Обнаружено изменение веса: %.1f г (предыдущий: %.1f г)\n", weight, lastWeight)

            var result string
            
            if state.Status.ArduinoConnected {
                // Полный режим: весы + Arduino
                devices.SendCommandToArduino(state.Arduino, config.CMD_GET_DIMENSIONS)
                length, width, height := devices.GetDimensionsFromArduino(state.Arduino)
                result = fmt.Sprintf("%.0f:%d:%d:%d", weight, length, width, height)
                state.Status.LastDimensions = result
                fmt.Println("📋 Результат (полные измерения):", result)
            } else {
                // Режим только весов
                result = fmt.Sprintf("%.0f", weight)
                state.Status.LastDimensions = result
                fmt.Println("📋 Результат (только вес):", result)
            }
            
            // Используем кроссплатформенную функцию для ввода
            err = simulateKeyPress(result)
            if err != nil {
                log.Printf("Ошибка симуляции ввода: %v", err)
                continue
            }

            // Добавляем задержку после успешного измерения
            fmt.Println("✅ Измерение завершено. Ожидание следующего объекта...")
            time.Sleep(3 * time.Second) // Увеличиваем задержку, чтобы избежать повторных измерений
            break
        }

        fmt.Println("Ожидание следующего объекта...")
        time.Sleep(2 * time.Second)
    }
}