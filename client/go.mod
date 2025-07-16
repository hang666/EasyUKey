module github.com/hang666/EasyUKey/client

go 1.24.5

replace github.com/hang666/EasyUKey/shared => ../shared

require (
	github.com/gorilla/websocket v1.5.3
	github.com/hang666/EasyUKey/shared v0.0.0
	github.com/labstack/echo/v4 v4.13.4
	github.com/yusufpapurcu/wmi v1.2.4
)

require (
	github.com/boombuler/barcode v1.0.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)
