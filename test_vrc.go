package main

import (
	"encoding/json"
	"fmt"

	"github.com/photoprism/photoprism/internal/meta"
)

func main() {
	jsonData := `[{
		"SourceFile": "VRChat_2026-05-02_11-21-58.859_2560x1440.png",
		"ExifToolVersion": 13.59,
		"WorldID": "wrld_75c07a93-423a-4b06-9f2a-716854479b97",
		"WorldDisplayName": "Polydance Studio",
		"AuthorID": "usr_394318d4-bdc4-467a-8f8a-b8a367310ad5",
		"Author": "Nixsoul"
	}]`

	var d meta.Data
	err := d.Exiftool([]byte(jsonData), "VRChat_2026-05-02_11-21-58.859_2560x1440.png")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	b, _ := json.MarshalIndent(d, "", "  ")
	fmt.Println(string(b))
}
