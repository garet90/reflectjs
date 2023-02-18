all: !example

!example:
	GOWASM=satconv,signext GOOS=js GOARCH=wasm go build -o ./bin/reflectjs.wasm ./example