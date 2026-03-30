package main

// Version is overridden at build time via -ldflags "-X main.Version=..."
var Version = "dev"

func main() {
	Execute()
}
