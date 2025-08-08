Betelgeuze-measure-system

README!
Система автоматического взвешивания и измерения габаритов.
Когда на весы ставят объект, на компьютер передается вес, а также отправляется команда с запросом габаритов на ардуино, ардуино считывает данные с датичокв, рассчитывает размеры и отправляет данные по uart порту на компьютер, на компьютере данные копируются через библиотеки в буфер обмена и автоматический вставляются в любое свободное поле
для установки необходимо перейти в корень проекта

УСТАНОВКА


Загрузить скетч arre/arre.ino в ардуино (должна быдь установлена arduino IDE)


Установить все зависимости и библиотеки для GO (должен быть установлен GO)


2.1)
go mod init "название папки в которой лежит mainofallV2"
go mod tidy
2.2) ИЛИ
go mod init "название папки в которой лежит mainofallV2"
go get github.com/atotto/clipboard
go get github.com/jacobsa/go-serial/serial
go get github.com/micmonay/keybd_event
go get go.bug.st/serial
go get go.bug.st/serial/enumerator

ЗАПУСК

3.1 вариант запуска через powershell без билда, 3.2 и 3.3 варианты запуска через exe файл, но для этого их нужно создать этими командами. Либо перетащить уже готовые. При 1 запуске брандмауэр попросит разрешения на запуск сервера, нужно с админ паролем подтвердить

3.1)запускаете go run mainofallV2.go (go должен быть установлен)
Генерация exe файла которые запускается в тихом режиме
3.2) go build -ldflags="-H=windowsgui" -o mainV2_silent.exe mainV2.go
Генерация exe файла который запускается в обычном режиме с окном
3.3) go build -o mainV2.exe mainV2.go

betelgeuze_reconnect.bat файл котрый закрывает предыдущий exe файл и запускает его заного (если не работает, то проверить путь в этом батнике и название файла)

5)в браузере: localhost:8080 для отладки. также в ардуино ide мониторингом сериал порта можно посмотреть какие датчики подключены, а какие нет.

когда на весы кладется вещь, датчики считывают размеры, а весы вес, передают на компьютер и копируют в буфер обмена а потом вставляют в любое пустое поле


Установка go через powershell

Скачиваем официальный установочный MSI файл (напр. версия 1.22.0)
Invoke-WebRequest -Uri "https://go.dev/dl/go1.22.0.windows-amd64.msi" -OutFile "$env:TEMP\go-installer.msi"
Start-Process msiexec.exe -Wait -ArgumentList '/i', "$env:TEMP\go-installer.msi", '/quiet', '/norestart'

Установка GOPATH
[Environment]::SetEnvironmentVariable("GOPATH", "$env:USERPROFILE\go", [EnvironmentVariableTarget]::Machine)

Добавление Go и GOPATH/bin в системный PATH
$goBinPath = "C:\Program Files\Go\bin"
goUserBinPath="goUserBinPath = "goUserBinPath="env:USERPROFILE\go\bin"
$existingPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
if (existingPath−notlike"∗existingPath -notlike "*existingPath−notlike"∗goBinPath*") {
newPath="newPath = "newPath="existingPath;goBinPath;goBinPath;goBinPath;goUserBinPath"
[Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::Machine)
}
после установки можно перезапустить Powershell и проверить go version

Сборка ардуино
на I2C справа где надписи G, V - это питание, слева C, D - это передача данных. Датчики подключаются на 0, 2, 4, 6 порт (на I2C это видно слева и справа около портов)
вот часть кода в ардуино которая считывает эти порты на I2C:
const byte sensors_pins[NUM_SENSORS] = {0, 2, 4, 6}; // LEFT, RIGHT, TOP, BACK




Ярлычки для простого запуска простыми пользователями:

Linux:
Чтобы файл betelgeuze.desktop начал работать на линуксе, пропиши следующие команды:
chmod +x Betelgeuze.desktop
gio set /home/user/Desktop/Betelgeuze.desktop metadata::trusted true


Windows:
На винде просто либо перетащи эти батники, либо создай ярылк на рабочем столе и ссылайся на эти батники. 
С ярлыком вариант поинтереснее, так как можно настроить картиночку на ярлычке
betelgeuze_reconnect.bat file is turning on script and reconecting script
betelgeuze_off.bat file is turning off the script


