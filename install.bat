@echo off

echo Installing...
powershell -Command "Invoke-WebRequest https://raw.githubusercontent.com/Konixy/create-app/master/bin/create-app.exe -OutFile c:\Programs\create-app\create-app.exe"

IF EXIST c:\Programs\create-app SET PATH=%PATH%;c:\Programs\create-app
echo Installed successfully!