package devices

import (
    "context"
    "errors"
    "fmt"
    "runtime"
    "strings"
    "sync"
    "time"
    
    "betelgeuze-measure-system-main/types"
    
    "go.bug.st/serial"
    "go.bug.st/serial/enumerator"
)

// getSerialPorts returns a list of available serial ports for the current platform
func getSerialPorts() ([]string, error) {
    ports, err := enumerator.GetDetailedPortsList()
    if err != nil {
        return nil, err
    }
    
    var portNames []string
    for _, port := range ports {
        portNames = append(portNames, port.Name)
    }
    
    // If no ports found via enumerator, fallback to platform-specific common ports
    if len(portNames) == 0 {
        portNames = getCommonPorts()
    }
    
    return portNames, nil
}

// getCommonPorts returns common serial port names based on the operating system
func getCommonPorts() []string {
    switch runtime.GOOS {
    case "windows":
        var ports []string
        for i := 1; i <= 20; i++ {
            ports = append(ports, fmt.Sprintf("COM%d", i))
        }
        return ports
    case "linux":
        return []string{
            "/dev/ttyUSB0", "/dev/ttyUSB1", "/dev/ttyUSB2", "/dev/ttyUSB3",
            "/dev/ttyACM0", "/dev/ttyACM1", "/dev/ttyACM2", "/dev/ttyACM3",
            "/dev/ttyS0", "/dev/ttyS1", "/dev/ttyS2", "/dev/ttyS3",
        }
    case "darwin": // macOS
        return []string{
            "/dev/cu.usbserial", "/dev/cu.usbmodem", 
            "/dev/tty.usbserial", "/dev/tty.usbmodem",
            "/dev/cu.SLAB_USBtoUART", "/dev/tty.SLAB_USBtoUART",
        }
    default:
        return []string{}
    }
}

// PortTestResult содержит результат тестирования порта
type PortTestResult struct {
    Port   *types.ScalePort
    Error  error
    PortName string
}

func ConnectToScale() (*types.ScalePort, error) {
    fmt.Println("🔍 Поиск весов на последовательных портах...")
    
    portNames, err := getSerialPorts()
    if err != nil {
        fmt.Printf("⚠️ Ошибка получения списка портов: %v\n", err)
        portNames = getCommonPorts()
    }
    
    // Дополнительная диагностика портов
    fmt.Println("📋 Доступные порты:")
    for _, name := range portNames {
        fmt.Printf("  - %s\n", name)
    }
    
    // Фильтруем неподходящие порты
    var validPorts []string
    for _, name := range portNames {
        // Skip non-existent common ports on Linux/macOS
        if runtime.GOOS != "windows" && strings.HasPrefix(name, "COM") {
            continue
        }
        validPorts = append(validPorts, name)
    }
    
    if len(validPorts) == 0 {
        return nil, errors.New("не найдено подходящих портов для проверки")
    }
    
    // Используем параллельную проверку портов
    return connectToScaleParallel(validPorts)
}

func connectToScaleParallel(portNames []string) (*types.ScalePort, error) {
    fmt.Printf("🚀 Начинаем параллельную проверку %d портов...\n", len(portNames))
    
    // Контекст с таймаутом для всей операции
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Канал для результатов
    resultChan := make(chan PortTestResult, len(portNames))
    
    // WaitGroup для отслеживания завершения всех горутин
    var wg sync.WaitGroup
    
    // Запускаем горутину для каждого порта
    for _, portName := range portNames {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            
            fmt.Printf("🔌 Начинаем проверку порта %s в отдельной горутине...\n", name)
            
            // Проверяем порт с контекстом
            port, err := testPortWithContext(ctx, name, 2)
            
            // Отправляем результат в канал
            select {
            case resultChan <- PortTestResult{Port: port, Error: err, PortName: name}:
            case <-ctx.Done():
                // Контекст отменен, закрываем соединение если оно было открыто
                if port != nil && port.Connection != nil {
                    port.Connection.Close()
                }
            }
        }(portName)
    }
    
    // Горутина для закрытия канала после завершения всех тестов
    go func() {
        wg.Wait()
        close(resultChan)
    }()
    
    // Ожидаем первый успешный результат или завершения всех тестов
    var lastError error
    successCount := 0
    errorCount := 0
    
    for result := range resultChan {
        if result.Error != nil {
            errorCount++
            lastError = result.Error
            fmt.Printf("  ❌ Ошибка на %s: %v\n", result.PortName, result.Error)
        } else if result.Port != nil {
            successCount++
            fmt.Printf("  ✅ Найдены весы на порту %s!\n", result.PortName)
            
            // Отменяем контекст, чтобы остановить остальные горутины
            cancel()
            
            // Закрываем все остальные соединения, которые могут прийти после
            go func() {
                for remainingResult := range resultChan {
                    if remainingResult.Port != nil && remainingResult.Port.Connection != nil {
                        remainingResult.Port.Connection.Close()
                    }
                }
            }()
            
            return result.Port, nil
        }
    }
    
    fmt.Printf("📊 Итоги проверки: успешных - %d, с ошибками - %d\n", successCount, errorCount)
    
    if lastError != nil {
        return nil, fmt.Errorf("весы не найдены ни на одном порту. Последняя ошибка: %v", lastError)
    }
    
    return nil, errors.New("весы не найдены ни на одном последовательном порту")
}

func testPortWithContext(ctx context.Context, name string, maxRetries int) (*types.ScalePort, error) {
    var lastErr error
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        // Проверяем, не отменен ли контекст
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        fmt.Printf("  📡 Попытка %d/%d открыть порт %s...\n", attempt, maxRetries, name)
        
        result, err := testPortWithContextInternal(ctx, name)
        if err == nil && result != nil {
            return result, nil
        }
        
        lastErr = err
        if err != nil {
            fmt.Printf("  ⚠️ Попытка %d неудачна: %v\n", attempt, err)
        }
        
        if attempt < maxRetries {
            // Используем контекст для прерывания ожидания
            select {
            case <-time.After(1 * time.Second):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    return nil, lastErr
}

func testPortWithContextInternal(ctx context.Context, name string) (*types.ScalePort, error) {
    fmt.Printf("  📡 Открываем порт %s...\n", name)
    
    configs := []struct {
        baudRate int
        dataBits int
        parity   serial.Parity
        stopBits serial.StopBits
        name     string
    }{
        {4800, 8, serial.EvenParity, serial.OneStopBit, "4800-8-E-1"},
        {9600, 8, serial.NoParity, serial.OneStopBit, "9600-8-N-1"},
        {2400, 8, serial.EvenParity, serial.OneStopBit, "2400-8-E-1"},
        {9600, 8, serial.EvenParity, serial.OneStopBit, "9600-8-E-1"},
    }
    
    for _, config := range configs {
        // Проверяем контекст перед каждой попыткой
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        fmt.Printf("  🔧 Пробуем конфигурацию %s...\n", config.name)
        
        mode := &serial.Mode{
            BaudRate: config.baudRate,
            DataBits: config.dataBits,
            StopBits: config.stopBits,
            Parity:   config.parity,
        }
        
        conn, err := serial.Open(name, mode)
        if err != nil {
            fmt.Printf("  ❌ Не удалось открыть с %s: %v\n", config.name, err)
            continue
        }
        
        // Тест коммуникации с контекстом
        result, testErr := testScaleCommunicationWithContext(ctx, conn, name, config.name)
        if result != nil {
            return result, nil
        }
        
        // Закрываем соединение только если тест не прошел
        conn.Close()
        if testErr != nil {
            fmt.Printf("  ❌ Тест с %s не прошел: %v\n", config.name, testErr)
        }
    }
    
    return nil, fmt.Errorf("все конфигурации не подошли для %s", name)
}

func testScaleCommunicationWithContext(ctx context.Context, conn serial.Port, portName, configName string) (*types.ScalePort, error) {
    // Проверяем контекст
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Set read timeout
    err := conn.SetReadTimeout(500 * time.Millisecond)
    if err != nil {
        return nil, fmt.Errorf("не удалось установить таймаут: %v", err)
    }
    
    // Очищаем буфер
    conn.SetReadTimeout(50 * time.Millisecond)
    buf := make([]byte, 256)
    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        n, err := conn.Read(buf)
        if err != nil || n == 0 {
            break
        }
    }
    conn.SetReadTimeout(500 * time.Millisecond)
    
    // Отправляем команду
    fmt.Printf("    📤 Отправляем команду 0x48 с конфигурацией %s...\n", configName)
    _, writeErr := conn.Write([]byte{0x48})
    if writeErr != nil {
        return nil, fmt.Errorf("ошибка записи: %v", writeErr)
    }
    
    // Ждем с проверкой контекста
    select {
    case <-time.After(300 * time.Millisecond):
    case <-ctx.Done():
        return nil, ctx.Err()
    }
    
    // Пробуем читать с проверкой контекста
    readBuf := make([]byte, 10)
    totalRead := 0
    
    for attempt := 0; attempt < 10; attempt++ {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        n, readErr := conn.Read(readBuf[totalRead:])
        if n > 0 {
            totalRead += n
            fmt.Printf("    📥 Получено %d байт (попытка %d)\n", n, attempt+1)
            
            if totalRead >= 2 {
                break
            }
        }
        
        if readErr != nil {
            if strings.Contains(readErr.Error(), "timeout") {
                if attempt < 5 {
                    fmt.Printf("    ⏰ Таймаут чтения (попытка %d)\n", attempt+1)
                }
                continue
            }
            return nil, fmt.Errorf("ошибка чтения: %v", readErr)
        }
        
        if n == 0 {
            select {
            case <-time.After(100 * time.Millisecond):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    if totalRead > 0 {
        fmt.Printf("    📥 Всего получено с %s (%s): %d байт - [", portName, configName, totalRead)
        for j := 0; j < totalRead; j++ {
            fmt.Printf("0x%02X", readBuf[j])
            if j < totalRead-1 {
                fmt.Print(", ")
            }
        }
        fmt.Printf("]\n")
        
        if totalRead >= 2 {
            fmt.Printf("    🔍 Анализ ответа: первый байт = %d (0x%02X), второй байт = %d (0x%02X)\n", 
                      readBuf[0], readBuf[0], readBuf[1], readBuf[1])
            
            validFirstByte := readBuf[0] == 128 || readBuf[0] == 192 || readBuf[0] == 160 || readBuf[0] == 224 || 
                             readBuf[0] == 144 || readBuf[0] == 176 || readBuf[0] == 208 || readBuf[0] == 240
            
            validSecondByte := totalRead >= 2 && readBuf[1] == 192
            
            if validFirstByte {
                fmt.Printf("    ✅ Найден валидный ответ от весов! Первый байт = %d (0x%02X)\n", readBuf[0], readBuf[0])
                return &types.ScalePort{Connection: conn, PortName: portName}, nil
            } else if validSecondByte {
                fmt.Printf("    ✅ Найден валидный ответ от весов! Второй байт = 192 (0xC0)\n")
                return &types.ScalePort{Connection: conn, PortName: portName}, nil
            }
        }
    } else {
        fmt.Printf("    📭 Нет данных от %s с %s\n", portName, configName)
    }
    
    return nil, errors.New("нет валидного ответа")
}

// ReadWeight читает вес с весов (оригинальная функция)
func ReadWeight(p *types.ScalePort) (float64, error) {
    _, err := p.Connection.Write([]byte{0x4A})
    if err != nil {
        return 0, fmt.Errorf("ошибка записи команды: %v", err)
    }
    
    time.Sleep(200 * time.Millisecond)
    buf := make([]byte, 5)
    n, err := p.Connection.Read(buf)
    if err != nil || n != 5 {
        return 0, errors.New("не удалось прочитать вес")
    }
    if buf[0] != 128 {
        return 0, nil
    }
    switch buf[1] {
    case 0:
        return float64(buf[3])*256 + float64(buf[2]), nil
    case 4:
        return (float64(buf[3])*256 + float64(buf[2])) * 10, nil
    default:
        return 0, nil
    }
}