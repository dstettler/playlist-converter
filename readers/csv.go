package readers

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

type csvFields struct {
	ArtistField      string
	AlbumArtistField string
	TitleField       string
	AlbumField       string
	TrackNumberField string
}

var fieldsTemplate = map[string]csvFields{
	"default": csvFields{
		ArtistField:      "Artist",
		AlbumArtistField: "AlbumArtist",
		TitleField:       "Title",
		AlbumField:       "Album",
		TrackNumberField: "Track Number",
	},
	"exportify": csvFields{
		ArtistField:      "Artist Name(s)",
		AlbumArtistField: "Album Artist Name(s)",
		TitleField:       "Track Name",
		AlbumField:       "Album Name",
		TrackNumberField: "Track Number",
	},
}

func ReadCsv(filename string, csvType string) PlaylistReader {
	csvFields := fieldsTemplate[csvType]

	ioReader, fileErr := os.Open(filename)
	if fileErr != nil {
		panic(fileErr)
	}

	reader := csv.NewReader(ioReader)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	csvReader := PlaylistReader{}
	artistIdx := -1
	albumArtistIdx := -1
	titleIdx := -1
	albumIdx := -1
	trackNumIdx := -1
	for i, record := range records {
		if i == 0 {
			for j, field := range record {
				if field == csvFields.ArtistField {
					artistIdx = j
				} else if field == csvFields.AlbumArtistField {
					albumArtistIdx = j
				} else if field == csvFields.TitleField {
					titleIdx = j
				} else if field == csvFields.AlbumField {
					albumIdx = j
				} else if field == csvFields.TrackNumberField {
					trackNumIdx = j
				}
			}

			if artistIdx == -1 &&
				albumArtistIdx == -1 &&
				titleIdx == -1 &&
				albumIdx == -1 &&
				trackNumIdx == -1 {
				panic("Input CSV does not include valid header")
			} else {
				// fmt.Println("Header indices:", artistIdx, albumArtistIdx, titleIdx, albumIdx, trackNumIdx)
			}
		} else {
			field := ReaderField{TrackNumber: -1}
			if artistIdx != -1 {
				field.Artist = record[artistIdx]
			}

			if albumArtistIdx != -1 {
				field.AlbumArtist = record[albumArtistIdx]
			}

			if titleIdx != -1 {
				field.Title = record[titleIdx]
			}

			if albumIdx != -1 {
				field.Album = record[albumIdx]
			}

			if trackNumIdx != -1 {
				field.TrackNumber, err = strconv.Atoi(record[trackNumIdx])
				if err != nil {
					fmt.Println("ERROR: Track number not valid integer")
				}

				field.TrackNumber = -1
			}

			csvReader.fields = append(csvReader.fields, field)
		}
	}

	return csvReader
}
