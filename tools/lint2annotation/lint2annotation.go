package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type diag struct {
	Category string `json:"category,omitempty"`
	Posn     string `json:"posn"`
	Message  string `json:"message"`
}

type jsonOutput map[string]map[string][]diag

func main() {
	outfile := os.Args[1]
	outBytes, err := os.ReadFile(outfile)
	if err != nil {
		log.Fatal(err)
	}

	out := jsonOutput{}
	if err := json.Unmarshal(outBytes, &out); err != nil {
		log.Fatal(err)
	}
	for _, pkg := range out {
		for _, diags := range pkg {
			for _, diag := range diags {
				pos := strings.Split(diag.Posn, ":")
				fmt.Printf("::error file=%s,line=%s,col=%s::%s\n", pos[0], pos[1], pos[2], diag.Message)
			}
		}
	}
}
