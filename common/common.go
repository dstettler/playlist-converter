package common

const AlbumArtistFormat = "AlbumArtist"
const AlbumFormat = "Album"
const ArtistFormat = "Artist"
const TitleFormat = "Title"
const TrackNumberFormat = "Track"

type MatchType int

const (
	Exact MatchType = iota
	ReverseMatch
	RegularMatch
)

type ConverterConfig struct {
	Paths []string
	Format string
	FormatMatchType MatchType
	MatchNum int
}

func MakeConverterConfig() ConverterConfig {
	return	ConverterConfig{
		Paths: nil,
		Format: ArtistFormat + "/" + AlbumFormat + "/" + TitleFormat,
		FormatMatchType: Exact,
		MatchNum: 0,
	}
}

type Song struct {
	Filepath    string
	Relpath string
	Title   string
	AlbumArtist string
	Artist      string
	Album       string
	TrackNumber int
}

func MakeSong() Song {
	return Song{}
}

