package web

import (
    "html/template"
    "net/http"
    
    "betelgeuze-measure-system-main/types"
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Система измерения веса и размеров</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .card { background: white; padding: 20px; margin: 10px 0; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .status { display: flex; justify-content: space-between; flex-wrap: wrap; }
        .device { flex: 1; min-width: 300px; margin: 5px; }
        .connected { color: #4CAF50; font-weight: bold; }
        .disconnected { color: #f44336; font-weight: bold; }
        button { background-color: #2196F3; color: white; border: none; padding: 10px 20px; margin: 5px; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #1976D2; }
        button:disabled { background-color: #ccc; cursor: not-allowed; }
        .commands { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 10px; }
        .command-group { border: 1px solid #ddd; padding: 15px; border-radius: 4px; }
        input, select { padding: 8px; margin: 5px; border: 1px solid #ddd; border-radius: 4px; }
        .response { background-color: #f0f0f0; padding: 10px; margin: 10px 0; border-radius: 4px; min-height: 50px; }
        .log { height: 200px; overflow-y: scroll; background-color: #000; color: #0f0; padding: 10px; font-family: monospace; font-size: 12px; }
        h1 { color: #333; text-align: center; }
        h2 { color: #555; border-bottom: 2px solid #2196F3; padding-bottom: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 Система измерения веса и размеров</h1>
        
        <div class="card">
            <h2>📊 Статус устройств</h2>
            <div class="status">
                <div class="device">
                    <h3>Arduino</h3>
                    <p>Статус: <span id="arduino-status" class="disconnected">Загрузка...</span></p>
                    <p>Порт: <span id="arduino-port">Загрузка...</span></p>
                </div>
                <div class="device">
                    <h3>Весы</h3>
                    <p>Статус: <span id="scale-status" class="disconnected">Загрузка...</span></p>
                    <p>Порт: <span id="scale-port">Загрузка...</span></p>
                </div>
                <div class="device">
                    <h3>Последние данные</h3>
                    <p>Вес: <span id="last-weight">-</span> г</p>
                    <p>Размеры: <span id="last-dimensions">-</span></p>
                </div>
            </div>
            <button onclick="reconnectDevices()">🔄 Переподключить устройства</button>
        </div>

        <div class="card">
            <h2>🎛️ Управление Arduino</h2>
            <div class="commands">
                <div class="command-group">
                    <h3>Базовые команды</h3>
                    <button onclick="sendArduinoCommand('start')">▶️ Старт</button>
                    <button onclick="sendArduinoCommand('ping')">🏓 Пинг</button>
                    <button onclick="sendArduinoCommand('reset_sensors')">🔄 Сброс сенсоров</button>
                    <button onclick="sendArduinoCommand('get_dimensions')">📏 Получить размеры</button>
                </div>
                
                <div class="command-group">
                    <h3>Светодиоды</h3>
                    <button onclick="sendArduinoCommand('led_on')">💡 Включить LED</button>
                    <button onclick="sendArduinoCommand('led_off')">⚫ Выключить LED</button>
                </div>
                
                <div class="command-group">
                    <h3>Настройка максимумов</h3>
                    <div>
                        <label>Высота:</label>
                        <input type="number" id="top-max" value="100" min="1" max="255">
                        <button onclick="setTopMax()">Установить</button>
                    </div>
                    <div>
                        <label>Ширина:</label>
                        <input type="number" id="width-max" value="100" min="1" max="255">
                        <button onclick="setWidthMax()">Установить</button>
                    </div>
                    <div>
                        <label>Длина:</label>
                        <input type="number" id="length-max" value="100" min="1" max="255">
                        <button onclick="setLengthMax()">Установить</button>
                    </div>
                </div>
            </div>
            
            <h3>Ответ Arduino:</h3>
            <div id="arduino-response" class="response">Ожидание команды...</div>
        </div>

        <div class="card">
            <h2>⚖️ Управление весами и измерениями</h2>
            <div style="display: flex; gap: 10px; flex-wrap: wrap; align-items: center;">
                <button onclick="readWeight()">📊 Считать только вес</button>
                <button onclick="combinedMeasure()" style="background-color: #4CAF50; font-weight: bold;">
                    🎯 Измерить ВСЁ + копировать
                </button>
            </div>
            
            <h3>Ответ системы:</h3>
            <div id="scale-response" class="response">Ожидание команды...</div>
        </div>

        <div class="card">
            <h2>📝 Лог системы</h2>
            <div id="system-log" class="log">Система запущена...\n</div>
        </div>
    </div>

    <script>
        function updateStatus() {
            fetch('/status')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('arduino-status').textContent = data.arduino_connected ? 'Подключен' : 'Отключен';
                    document.getElementById('arduino-status').className = data.arduino_connected ? 'connected' : 'disconnected';
                    document.getElementById('arduino-port').textContent = data.arduino_port;
                    
                    document.getElementById('scale-status').textContent = data.scale_connected ? 'Подключен' : 'Отключен';
                    document.getElementById('scale-status').className = data.scale_connected ? 'connected' : 'disconnected';
                    document.getElementById('scale-port').textContent = data.scale_port;
                    
                    document.getElementById('last-weight').textContent = data.last_weight || '-';
                    document.getElementById('last-dimensions').textContent = data.last_dimensions || '-';
                })
                .catch(err => {
                    addLog('Ошибка получения статуса: ' + err);
                });
        }

        // Подключение к потоку логов
        function connectToLogs() {
            const eventSource = new EventSource('/logs/stream');
            
            eventSource.onmessage = function(event) {
                const logData = JSON.parse(event.data);
                addLogToDisplay(logData.time, logData.message, logData.type);
            };
            
            eventSource.onerror = function(event) {
                console.error('Ошибка подключения к логам:', event);
                addLogToDisplay(new Date().toLocaleTimeString(), 'Ошибка подключения к логам, переподключение через 5 сек...', 'system');
                setTimeout(connectToLogs, 5000);
            };
        }
        
        function addLogToDisplay(time, message, type) {
            const log = document.getElementById('system-log');
            const colorMap = {
                'arduino': '#00ff00',
                'scale': '#ffff00', 
                'system': '#ffffff'
            };
            const color = colorMap[type] || '#ffffff';
            
            const logEntry = document.createElement('div');
            logEntry.style.color = color;
            logEntry.textContent = '[' + time + '] [' + type.toUpperCase() + '] ' + message;
            
            log.appendChild(logEntry);
            log.scrollTop = log.scrollHeight;
            
            // Ограничиваем количество строк в логе
            while (log.children.length > 1000) {
                log.removeChild(log.firstChild);
            }
        }

        function reconnectDevices() {
            addLog('Переподключение устройств...');
            fetch('/reconnect', {method: 'POST'})
                .then(response => response.text())
                .then(data => {
                    addLog('Результат переподключения: ' + data);
                    updateStatus();
                })
                .catch(err => {
                    addLog('Ошибка переподключения: ' + err);
                });
        }

        function sendArduinoCommand(cmd) {
            addLog('Отправка команды Arduino: ' + cmd);
            fetch('/arduino/command', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({command: cmd})
            })
            .then(response => response.text())
            .then(data => {
                document.getElementById('arduino-response').textContent = data;
                addLog('Ответ Arduino: ' + data);
            })
            .catch(err => {
                document.getElementById('arduino-response').textContent = 'Ошибка: ' + err;
                addLog('Ошибка команды Arduino: ' + err);
            });
        }

        function setTopMax() {
            const value = document.getElementById('top-max').value;
            sendArduinoCommand('set_top_max:' + value);
        }

        function setWidthMax() {
            const value = document.getElementById('width-max').value;
            sendArduinoCommand('set_width_max:' + value);
        }

        function setLengthMax() {
            const value = document.getElementById('length-max').value;
            sendArduinoCommand('set_length_max:' + value);
        }

        function readWeight() {
            addLog('Считывание веса...');
            fetch('/scale/read', {method: 'POST'})
                .then(response => response.text())
                .then(data => {
                    document.getElementById('scale-response').textContent = data;
                    addLog('Вес: ' + data);
                })
                .catch(err => {
                    document.getElementById('scale-response').textContent = 'Ошибка: ' + err;
                    addLog('Ошибка чтения веса: ' + err);
                });
        }

        function combinedMeasure() {
            addLog('Выполняется комплексное измерение...');
            const button = event.target;
            button.disabled = true;
            button.textContent = '⏳ Измеряю...';
            
            fetch('/measure/combined', {method: 'POST'})
                .then(response => response.text())
                .then(data => {
                    document.getElementById('scale-response').textContent = data;
                    addLog('Комплексное измерение: ' + data);
                    
                    // Показываем уведомление об успешном копировании
                    const notification = document.createElement('div');
                    notification.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 1000; background: #4CAF50; color: white; padding: 15px 20px; border-radius: 5px; box-shadow: 0 2px 10px rgba(0,0,0,0.3); font-weight: bold;';
                    notification.textContent = '✅ Данные скопированы в буфер обмена!';
                    document.body.appendChild(notification);
                    
                    setTimeout(function() {
                        notification.remove();
                    }, 3000);
                })
                .catch(err => {
                    document.getElementById('scale-response').textContent = 'Ошибка: ' + err;
                    addLog('Ошибка комплексного измерения: ' + err);
                })
                .finally(() => {
                    button.disabled = false;
                    button.textContent = '🎯 Измерить ВСЁ + копировать';
                });
        }        

        function addLog(message) {
            const log = document.getElementById('system-log');
            const time = new Date().toLocaleTimeString();
            log.textContent += '[' + time + '] ' + message + '\n';
            log.scrollTop = log.scrollHeight;
        }

        // Обновляем статус каждые 2 секунды
        setInterval(updateStatus, 2000);
        updateStatus();
        // Подключаемся к потоку логов
        connectToLogs();
    </script>
</body>
</html>
`

func indexHandler(w http.ResponseWriter, r *http.Request, state *types.AppState) {
    t, _ := template.New("index").Parse(htmlTemplate)
    t.Execute(w, state)
}