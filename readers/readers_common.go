package readers

import (
	"strconv"
	"strings"

	common "dstet.me/p2m3u/common"
)

type ReaderField struct {
	Title       string
	AlbumArtist string
	Album       string
	Artist      string
	TrackNumber int
}

type PlaylistReader struct {
	fields []ReaderField
}

func (r PlaylistReader) GetKeyList(format string) []string {
	var keys []string

	splitFormat := strings.Split(format, "/")
	for _, field := range r.fields {
		var key strings.Builder
		for i, ident := range splitFormat {
			var identVal string

			if ident == common.AlbumFormat {
				identVal = field.Album
			} else if ident == common.AlbumArtistFormat {
				identVal = field.AlbumArtist
			} else if ident == common.ArtistFormat {
				identVal = field.Artist
			} else if ident == common.TitleFormat {
				identVal = field.Title
			} else if ident == common.TrackNumberFormat {
				identVal = strconv.Itoa(field.TrackNumber)
			}

			if identVal == "" {
				identVal = "Unknown"
			}

			key.WriteString(identVal)

			// Only append '/' on nonfinal idents
			if i < len(splitFormat)-1 {
				key.WriteString("/")
			}
		}

		keys = append(keys, key.String())
	}

	return keys
}
