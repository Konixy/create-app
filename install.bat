@echo off

echo Installing...

set dest="c:\Program Files\create-app"

mkdir %dest%
cd %dest%
powershell -Command Invoke-WebRequest https://raw.githubusercontent.com/Konixy/create-app/master/bin/create-app.exe -OutFile ./create-app.exe

IF EXIST %dest% SET PATH=%PATH%;c:\Program Files\create-app

echo Installed successfully!
pause