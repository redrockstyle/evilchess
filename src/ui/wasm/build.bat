@echo off

set GOOS=js
set GOARCH=wasm
go build -o main.wasm main.go

for /f "usebackq delims=" %%i in (`go env GOROOT`) do set GOROOT=%%i
copy "%GOROOT%\lib\wasm\wasm_exec.js" .

echo Start 'python3 -m http.server 8080' and go to http://localhost:8080/index.html