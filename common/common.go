package common

const AlbumArtistFormat = "AlbumArtist"
const AlbumFormat = "Album"
const ArtistFormat = "Artist"
const TitleFormat = "Title"
const TrackNumberFormat = "Track"

type MatchType int

const (
	Exact MatchType = iota
	FuzzyMatch
)

type ConverterConfig struct {
	Paths           []string
	Format          string
	FormatMatchType MatchType
	FuzzyMatchNum   int
}

func MakeConverterConfig() ConverterConfig {
	return ConverterConfig{
		Paths:           nil,
		Format:          ArtistFormat + "/" + AlbumFormat + "/" + TitleFormat,
		FormatMatchType: Exact,
		FuzzyMatchNum:   0,
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
