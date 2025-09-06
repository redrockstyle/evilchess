$env:GOOS="js"; $env:GOARCH="wasm"; go build -o main.wasm main.go

Copy-Item -Path "$(go env GOROOT)\lib\wasm\wasm_exec.js" -Destination .

Write-Output "Start 'python3 -m http.server 8080' and go to http://localhost:8080/index.html"
