package main

import (
	"os"

	"github.com/thaoanhhaa1/ghm/internal/app"
)

func main() {
	os.Exit(app.Main(os.Args[1:]))
}
