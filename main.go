package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func usage() {
	binName := filepath.Base(os.Args[0])
	fmt.Println(strings.TrimSpace(fmt.Sprintf(`
Usage:
	Server: %s server -local :6000 -public :5000 -secret "KEY"
	Client: %s client -server VPS_IP:5000 -local 127.0.0.1:10808 -secret "KEY"
`, binName, binName)))
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		fs := flag.NewFlagSet("server", flag.ExitOnError)
		localAddr := fs.String("local", "", "listening address for public clients")
		publicAddr := fs.String("public", "", "listening address for tunnel connections")
		secret := fs.String("secret", "", "shared secret key")
		_ = fs.Parse(os.Args[2:])

		if *localAddr == "" || *publicAddr == "" || *secret == "" {
			log.Fatal("Error: missing required flags. -local, -public, and -secret are all mandatory.")
		}

		if err := runServer(*localAddr, *publicAddr, *secret); err != nil {
			log.Fatal(err)
		}

	case "client":
		fs := flag.NewFlagSet("client", flag.ExitOnError)
		serverAddr := fs.String("server", "", "server tunnel address")
		localAddr := fs.String("local", "", "local service address")
		secret := fs.String("secret", "", "shared secret key")
		_ = fs.Parse(os.Args[2:])

		if *serverAddr == "" || *localAddr == "" || *secret == "" {
			log.Fatal("Error: missing required flags. -server, -local, and -secret are all mandatory.")
		}

		if err := runClient(*serverAddr, *localAddr, *secret); err != nil {
			log.Fatal(err)
		}

	default:
		usage()
		os.Exit(1)
	}
}
