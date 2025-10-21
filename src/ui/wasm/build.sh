#!/bin/bash

GOOS=js GOARCH=wasm go build -o main.wasm main.go
cp $(go env GOROOT)/lib/wasm/wasm_exec.js .

echo "Start 'python3 -m http.server 8080' and go to http://localhost:8080/index.html"