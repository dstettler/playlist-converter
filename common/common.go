package common

import (
	"archive/zip"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const AlbumArtistFormat = "AlbumArtist"
const AlbumFormat = "Album"
const ArtistFormat = "Artist"
const TitleFormat = "Title"
const TrackNumberFormat = "Track"
const FormatSeparatorCharacter = "\u001E"

const ArtistMatchVal = 0.3
const AlbumArtistMatchVal = 0.3
const AlbumMatchVal = 0.5
const TitleMatchVal = 0.1

const UnknownArtist = "Unknown Artist"

const ConverterDbFile = "converter.db"
const ZippedFilename = "zipped"

var filetypeDefaultBonuses = map[string]float32{
	"OGG":  0,
	"MP3":  0,
	"FLAC": 0.2,
	"M4A":  0.1,
}

type ConverterConfig struct {
	Paths                 []string
	Format                string
	MinimumMatchAllowance float32
	FiletypeBonuses       map[string]float32
	SplitCharacter        string
	SpecialCases          []string
}

func MakeConverterConfig() ConverterConfig {
	return ConverterConfig{
		Paths:                 nil,
		Format:                ArtistFormat + FormatSeparatorCharacter + AlbumFormat + FormatSeparatorCharacter + TitleFormat,
		MinimumMatchAllowance: 0.9,
		FiletypeBonuses:       filetypeDefaultBonuses,
		SplitCharacter:        ",",
		SpecialCases:          nil,
	}
}

type Song struct {
	Filepath    string
	Relpath     string
	Title       string
	AlbumArtist string
	Artist      string
	Album       string
	TrackNumber int
}

func MakeSong() Song {
	return Song{}
}

type ConverterLibrary struct {
	// Map of internal song id to Song struct.
	Songs map[int]*Song
	// Map of Artists to list of songs.
	ArtistsIndex map[string][]int
	// Map of Album Artists to list of songs.
	AlbumArtistsIndex map[string][]int
	// Map of Albums to list of songs.
	AlbumsIndex map[string][]int
	// Map of Titles to list of songs.
	TitlesIndex map[string][]int
	// Song ids for indices.
	Ids map[string]int
	// Next id for id list.
	NextId int
}

func MakeLibrary() ConverterLibrary {
	return ConverterLibrary{
		Songs:             make(map[int]*Song),
		ArtistsIndex:      make(map[string][]int),
		AlbumsIndex:       make(map[string][]int),
		AlbumArtistsIndex: make(map[string][]int),
		TitlesIndex:       make(map[string][]int),
		Ids:               make(map[string]int),
		NextId:            0,
	}
}

// Reads ConverterLibrary from file specified.
func (lib *ConverterLibrary) TryReadDbFile(file string) {
	if file == "" {
		file = ConverterDbFile
	}

	if _, err := os.Stat(file); err == nil {
		fmt.Println("Existing db file found. Reading...")
		zipR, err := zip.OpenReader(file)
		if err != nil {
			panic(err)
		}

		gobFile, err := zipR.Open(ZippedFilename)
		if err != nil {
			panic(err)
		}

		decoder := gob.NewDecoder(gobFile)
		decoder.Decode(&lib)
		gobFile.Close()
		zipR.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}
}

// Writes ConverterLibrary to file specified.
func (lib ConverterLibrary) WriteDbFile(file string) {
	if file == "" {
		file = ConverterDbFile
	}

	zipFile, err := os.Create(file)
	if err != nil {
		panic(err)
	}

	zipWriter := zip.NewWriter(zipFile)

	gobFile, err := zipWriter.Create(ZippedFilename)
	if err != nil {
		panic(err)
	}
	encoder := gob.NewEncoder(gobFile)
	encoder.Encode(lib)

	zipWriter.Close()
	zipFile.Close()
}

// Returns id of a song, otherwise returns -1.
func (lib ConverterLibrary) GetId(path string) int {
	if id, exists := lib.Ids[path]; exists {
		return id
	} else {
		return -1
	}
}

// Returns new id for a new song.
func (lib *ConverterLibrary) GetNewId(path string) int {
	id := lib.NextId
	lib.Ids[path] = id

	lib.NextId += 1

	return id
}

// Helper function to return a list of possible matches.
func (lib ConverterLibrary) getMatchCandidates(formatStr string, config *ConverterConfig) map[int]float32 {
	// Use a map in place of a set (to avoid dupes).
	candidateMap := make(map[int]float32)
	splitFormatStr := strings.Split(formatStr, FormatSeparatorCharacter)

	for i, split := range strings.Split(config.Format, FormatSeparatorCharacter) {
		if split == ArtistFormat {
			// Special case for artists, since there may be multiple.
			for _, splitArtist := range ArtistSplit(splitFormatStr[i], config) {
				trimmedArtist := strings.TrimSpace(splitArtist)
				for _, candidate := range lib.ArtistsIndex[trimmedArtist] {
					if val, present := candidateMap[candidate]; present {
						candidateMap[candidate] = val + ArtistMatchVal
					} else {
						candidateMap[candidate] = ArtistMatchVal
					}
				}
			}
		} else if split == AlbumArtistFormat {
			// Again, special case for artists, since there may be multiple.
			for _, splitArtist := range ArtistSplit(splitFormatStr[i], config) {
				for _, candidate := range lib.AlbumArtistsIndex[strings.TrimSpace(splitArtist)] {
					if val, present := candidateMap[candidate]; present {
						candidateMap[candidate] = val + AlbumArtistMatchVal
					} else {
						candidateMap[candidate] = AlbumArtistMatchVal
					}
				}
			}
		} else if split == AlbumFormat {
			for _, candidate := range lib.AlbumsIndex[splitFormatStr[i]] {
				if val, present := candidateMap[candidate]; present {
					candidateMap[candidate] = val + AlbumMatchVal
				} else {
					candidateMap[candidate] = AlbumMatchVal
				}
			}
		} else if split == TitleFormat {
			for _, candidate := range lib.TitlesIndex[splitFormatStr[i]] {
				if val, present := candidateMap[candidate]; present {
					candidateMap[candidate] = val + TitleMatchVal
				} else {
					candidateMap[candidate] = TitleMatchVal
				}
			}
		}
	}

	return candidateMap
}

// Function to get a Song ptr based on format string and ConverterConfig allowances.
func (lib ConverterLibrary) GetSongFromFormatString(formatStr string, config *ConverterConfig) *Song {
	candidates := lib.getMatchCandidates(formatStr, config)

	var greatestVal float32
	greatestVal = -1.0
	currentCandidate := -1
	for candidate, val := range candidates {
		// Add any additional values based on the candidate (this can positively bias
		// a specific version of a file in the case of dupes)
		ext := GetFileExtension(lib.Songs[candidate].Filepath)
		val += config.FiletypeBonuses[strings.ToUpper(ext)]

		if val > greatestVal {
			currentCandidate = candidate
			greatestVal = val
		}
	}

	if greatestVal > config.MinimumMatchAllowance {
		return lib.Songs[currentCandidate]
	} else {
		return nil
	}
}

// Returns a capitalized version of a given file extension from a file.
// If the file does not include any dots, will return filename as-is.
func GetFileExtension(filename string) string {
	if strings.Contains(filename, ".") {
		splitStr := strings.Split(filename, ".")
		return strings.ToUpper(splitStr[len(splitStr)-1])
	} else {
		return filename
	}
}

// Returns true if val is found in any of the multimatch ranges.
func matchInMultimatch(val int, multiMatch [][]int) bool {
	for _, match := range multiMatch {
		if val >= match[0] && val < match[1] {
			return true
		}
	}

	return false
}

// Splits on the library's dedicated split character.
func ArtistSplit(artists string, config *ConverterConfig) []string {
	var containsSpecial []string
	for _, special := range config.SpecialCases {
		if strings.Contains(artists, special) {
			containsSpecial = append(containsSpecial, special)
		}
	}

	// If the string does not contain a special case that needs to be ignored
	if len(containsSpecial) < 1 {
		// Some formats will have an escape char in the artist name
		re := regexp.MustCompile(`[^\\]` + config.SplitCharacter)

		matches := re.FindAllStringIndex(artists, -1)
		var split []string
		for i, match := range matches {
			if i == 0 {
				split = append(split, artists[0:match[1]-1])
			} else {
				split = append(split, artists[matches[i-1][1]:match[1]-1])
			}
		}

		if len(matches) > 0 {
			split = append(split, artists[matches[len(matches)-1][1]:])
		} else {
			split = append(split, artists)
		}

		// Replace instances of the escaped split char with just the char itself
		for i := range split {
			split[i] = strings.ReplaceAll(split[i], "\\"+config.SplitCharacter, config.SplitCharacter)
		}

		return split
	} else {
		// Since there are special characters we need to ignore any matches found in their ranges
		baseRe := regexp.MustCompile(`[^\\]` + config.SplitCharacter)

		var multiArtistReString strings.Builder
		for i, special := range containsSpecial {
			if i != len(containsSpecial)-1 {
				multiArtistReString.WriteString(special + "|")
			} else {
				multiArtistReString.WriteString(special)
			}
		}

		multiArtistRe := regexp.MustCompile(multiArtistReString.String())

		baseMatches := baseRe.FindAllStringIndex(artists, -1)
		multiArtistMatches := multiArtistRe.FindAllStringIndex(artists, -1)

		var split []string
		for i, match := range baseMatches {
			if !matchInMultimatch(match[1]-1, multiArtistMatches) {
				if len(split) == 0 {
					split = append(split, artists[0:match[1]-1])
				} else {
					split = append(split, artists[baseMatches[i-1][1]:match[1]-1])
				}
			}
		}

		if len(baseMatches) > len(multiArtistMatches) {
			split = append(split, artists[baseMatches[len(baseMatches)-1][1]:])
		} else {
			split = append(split, artists)
		}

		// Replace instances of the escaped split char with just the char itself
		for i := range split {
			split[i] = strings.ReplaceAll(split[i], "\\"+config.SplitCharacter, config.SplitCharacter)
		}

		return split
	}
}
