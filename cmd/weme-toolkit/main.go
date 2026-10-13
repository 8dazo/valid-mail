// weme-toolkit exposes the same local Go tools as a JSON-lines CLI.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/8dazo/valid-mail/internal/toolkit"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request toolkit.Request
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			_ = encoder.Encode(toolkit.Response{Error: "Invalid tool request."})
			continue
		}
		if err := encoder.Encode(toolkit.Execute(request)); err != nil {
			fmt.Fprintln(os.Stderr, "Could not encode tool output.")
			os.Exit(1)
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintln(os.Stderr, "Input exceeds the CLI request limit.")
		os.Exit(1)
	}
}
