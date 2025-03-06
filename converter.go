package main

import (
	"os"
	"errors"
	"fmt"
	"log"
	"go.senan.xyz/taglib"
	"./lib"

)
import "github.com/alecthomas/kong"
import "github.com/pelletier/go-toml/v2"

type ConverterConfig struct {
	Paths []string
	Format string
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
	// @TODO
	return ""
}

func readSong(filepath string) Song {
	song := MakeSong()
	return song
}

func addSongsRecursive(dir string, reldir string, songs map[string]Song) {
	if dirItems, err := os.ReadDir(dir); err != nil {
		for _, item := range dirItems {
			if (item.IsDir()) {
				newDir := osPathJoin(dir, item.Name())
				newRel := reldir + "/" + item.Name()
				addSongsRecursive(newDir, newRel, songs)
			}
		}
	}
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

	for _, path := range config.Paths {
		addSongsRecursive(path, "", foundSongs)
	}
			
	tags, err := taglib.ReadTags("/home/devon/Music/1_1_Catharsis.ogg")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tags: %v\n", tags) // map[string][]string

	fmt.Printf("AlbumArtist: %q\n", tags[taglib.AlbumArtist])
	fmt.Printf("Album: %q\n", tags[taglib.Album])
	fmt.Printf("TrackNumber: %q\n", tags[taglib.TrackNumber])
}
