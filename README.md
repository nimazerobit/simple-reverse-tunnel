# Simple Reverse Tunnel - Multiplexed Reverse Tunnel

Simple Reverse Tunnel is a lightweight, secure reverse tunnel application written in Go.

It uses `github.com/hashicorp/yamux` to multiplex multiple connections over a single, persistent, and secure TCP tunnel.

## Features
* Multiplexing: Handles multiple concurrent connections over a single TCP tunnel using Yamux.
* Security: Custom handshake with a shared secret key.
* Reconnect: Automatic reconnection on the client side if the tunnel drops.
* Simplicity: Single binary acting as both server and client.

## Prerequisites
*   Go 1.18 or higher
*   `github.com/hashicorp/yamux`

## Build Instructions
1. Clone the repository.
```shell
git clone https://github.com/nimazerobit/simple-reverse-tunnel.git
cd simple-reverse-tunnel
```
2. Download the required dependencies:
```
go get github.com/hashicorp/yamux
```
3. Build the executable:
- For Windows:
```
go build -o tunnel.exe
```
- For Linux:
```
go build -o tunnel
```
- Compile for Linux on Windows:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o tunnel
```

## How to use it?
> [!NOTE]
> All command-line arguments are mandatory.

1. On Server
   The server needs to listen on two ports: one for incoming connections (-local), and one for the tunnel connection (-public).
   
   ./tunnel server -local :6000 -public :5000 -secret "YOUR_SECRET_KEY"

   -local : Address to listen on for incoming traffic.
   -public: Address to listen on for the tunnel connection.
   -secret: The shared secret key for authenticating the tunnel.

2. On Client
   The client connects to the server's tunnel address and forwards traffic to your local service.

   ./tunnel client -server VPS_IP:5000 -local 127.0.0.1:10808 -secret "YOUR_SECRET_KEY"

   -server: The address of the server's tunnel listener (matches server's -public flag).
   -local : The address of the local service you want to expose (e.g., v2ray proxy on your client machine).
   -secret: The shared secret key must match the server's secret.

> [!NOTE]
> Put simply, this configuration exposes your local port 10808 to the outside world via the remote server’s port 6000, without requiring any router changes, port forwarding, or firewall configurations from your ISP.
