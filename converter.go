package main

import (
	"os"
	"path/filepath"
	"errors"
	"fmt"
	"strings"
	"slices"
	"io/fs"
	"strconv"
	"log"
	"runtime"
	"go.senan.xyz/taglib"
	readers "dstet.me/p2m3u/readers"

)
import "github.com/alecthomas/kong"
import "github.com/pelletier/go-toml/v2"

var WHITELISTED_FILETYPES = []string {
	"OGG",
	"MP3",
	"M4A",
	"FLAC",
	"WAV",
	"AIFF",
}

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
		Format: "AlbumArtist/Album/Title",
	}
}

type Song struct {
	Filepath    string
	Relpath string
	SongTitle   string
	AlbumArtist string
	Artist      string
	Album       string
	TrackNumber int
}

func MakeSong() Song {
	return Song{}
}


var CLI struct {
	Input string `arg:"" help:"Input playlist" type:"path"`	
	Output string `arg:"" help:"Output file" type:"path"`
	SearchDirs  []string `arg:"" help:"Directories to search" type:"path" optional:""`
	Config string `short:"c" help:"Config file to use" type:"path"`
	Verbosity int    `type:"counter" short:"v" optional:"" help:"Verbosity counter" default:"0"`
}

func parseConfig(filepath string) ConverterConfig {
	if _, err  := os.Stat(filepath); err == nil {
		var config ConverterConfig	
		if fileContents, fileErr := os.ReadFile(filepath); fileErr != nil {
			fmt.Println("ERROR: Error reading config file:", err, "Using default configuration")
			return MakeConverterConfig()
		} else {
			if err := toml.Unmarshal(fileContents, &config); err != nil {
				panic(err)
			}
			fmt.Println("Unmarshaled:", config, "From:", fileContents)
			if len(config.Paths) < 1 {
				config.Paths = nil
			}
			return config
		}
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Println("ERROR: Specified config file does not exist:", filepath, "Using default configuration")
		return MakeConverterConfig()
	} else {
		panic(err)
	}	
}

func osPathJoin(path1 string, path2 string) string {
	if runtime.GOOS == "windows" {
		return path1 + "\\" + path2
	} else {
		return path1 + "/" + path2
	}
}

func readSong(filepath string, relpath string) Song {
	song := MakeSong()
	fmt.Println("Reading:", )
	// return song
	tags, err := taglib.ReadTags(filepath)
	if err != nil {
		log.Fatal(err)
		return song
	}

	if len(tags[taglib.Album]) > 0 {
		song.Album = tags[taglib.Album][0]
	}
	
	if len(tags[taglib.AlbumArtist]) > 0 {
		song.AlbumArtist = tags[taglib.AlbumArtist][0]
	}

	if len(tags[taglib.Title]) > 0 {
		song.SongTitle = tags[taglib.Title][0]
	}

	if len(tags[taglib.TrackNumber]) > 0 {
		song.TrackNumber, _ = strconv.Atoi(tags[taglib.TrackNumber][0])
	}

	song.Filepath = filepath
	song.Relpath = relpath

	return song
}

func getFileExtension(filename string) string {
	if strings.Contains(filename, ".") {
		splitStr := strings.Split(filename, ".")
		return splitStr[1]
	} else {
		return filename
	}
}

func addSongsRecursive(dir string, reldir string, songs map[string]Song) {
	fileSystem := os.DirFS(dir)
	fs.WalkDir(fileSystem, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.Type().IsDir() {	
			ext := getFileExtension(dirEntry.Name())
			if ext != dirEntry.Name() && slices.Contains(WHITELISTED_FILETYPES, strings.ToUpper(ext)) {
				key := reldir + "/" + path
				filepath := osPathJoin(dir, path)
				songs[key] = readSong(filepath, key)
			}
		}
		return nil
	})
}

func main() {
	kong.Parse(&CLI, kong.Description("A utility that takes in a CSV playlist of song metadata and converts it to a relative-pathed M3U playlist."))
	var config ConverterConfig
	foundSongs := make(map[string]Song)

	if (CLI.Config != "") {
		config = parseConfig(CLI.Config)
	} else {
		config = MakeConverterConfig()
	}

	if len(CLI.SearchDirs) > 0 && config.Paths == nil {
		config.Paths = CLI.SearchDirs
	} else if len(CLI.SearchDirs) > 0 {
		config.Paths = append(config.Paths, CLI.SearchDirs...)
	}

	if (config.Paths == nil) {
		fmt.Println("ERROR: No search paths specified! Unable to continue.")
		return
	}

	fmt.Println("Building database...")
	for _, path := range config.Paths {
		fmt.Println("Reading", path)
		addSongsRecursive(path, filepath.Base(path), foundSongs)
	}

	fmt.Println(foundSongs)
}
