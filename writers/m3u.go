package writers

import (
	"os"
	"strings"

	common "dstet.me/p2m3u/common"
)

func WriteM3U(filename string, list []*common.Song) {
	var builder strings.Builder
	for _, song := range list {
		if song != nil {
			builder.WriteString(song.Relpath)
			builder.WriteString("\n")
		}
	}

	outputFile, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	outputFile.WriteString(builder.String())
	outputFile.Close()
}
