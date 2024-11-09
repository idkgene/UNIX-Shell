package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	for {
						// Print prompt
						fmt.Print("> ")

						// Read user input
						var input string
						fmt.Scanln(&input)

						// Parse input
						args := strings.Fields(input)
						if len(args) == 0 {
							continue
						}

						switch args[0] {
						case "exit":
							os.Exit(0)
						case "cd": 
						if len(args) < 2 {
							fmt.Println("Path required for cd")
							continue
						}
						os.Chdir(args[1])
						continue
						}
						cmd := exec.Command(args[0], args[1:]...)
						cmd.Stderr = os.Stderr
						cmd.Stdout = os.Stdout
						err := cmd.Run()
						
						if err != nil {
							fmt.Println("Error:", err)
						}
	}
}
