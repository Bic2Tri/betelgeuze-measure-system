@echo off
setlocal

set EXE_NAME=mainV2_silent.exe
set EXE_PATH=\\192.168.10.5\scan\betelgeuze-measure-system-measure-V4\%EXE_NAME%
set EXE_PATH2=%USERPROFILE%\Desktop\betelgeuze-measure-system-main\%EXE_NAME%

REM Проверяем, запущен ли процесс
tasklist /FI "IMAGENAME eq %EXE_NAME%" | find /I "%EXE_NAME%" >nul
if %ERRORLEVEL%==0 (
    echo Процесс %EXE_NAME% найден. Завершаем...
    taskkill /IM "%EXE_NAME%" /F >nul
    echo Завершено.
) else (
    echo Процесс %EXE_NAME% не найден.
)

endlocal