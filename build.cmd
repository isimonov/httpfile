if not exist "./build" mkdir ./build

set GOOS=linux
set GOARCH=amd64
go build -o ./build/httpfile .

set GOOS=windows
set GOARCH=amd64
go build -o ./build/httpfile.exe .