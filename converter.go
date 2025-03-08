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
	"encoding/gob"
	"runtime"
	"go.senan.xyz/taglib"
	readers "dstet.me/p2m3u/readers"
	writers "dstet.me/p2m3u/writers"
	common "dstet.me/p2m3u/common"
)
import "github.com/alecthomas/kong"
import "github.com/pelletier/go-toml/v2"

var WhitelistedFiletypes = []string {
	"OGG",
	"MP3",
	"M4A",
	"FLAC",
	"WAV",
	"AIFF",
}

var InputTypes = []string {
	"CSV",
	"EXPORTIFY",
}

var OutputTypes = []string {
	"M3U",
}
const ConverterDbFile = "converter.db"

var CLI struct {
	Input string `arg:"" help:"Input playlist" type:"path"`	
	Output string `arg:"" help:"Output file" type:"path"`
	SearchDirs  []string `arg:"" help:"Directories to search" type:"path" optional:""`
	Config string `short:"c" help:"Config file to use" type:"path"`
	InputType string `short:"i" help:"Mode to parse input file" optional:""`
	OutputType string `short:"o" help:"Mode to write output file" optional:""`
	Verbosity int    `type:"counter" short:"v" optional:"" help:"Verbosity counter" default:"0"`
}

func parseConfig(filepath string) common.ConverterConfig {
	if _, err  := os.Stat(filepath); err == nil {
		var config common.ConverterConfig	
		if fileContents, fileErr := os.ReadFile(filepath); fileErr != nil {
			fmt.Println("ERROR: Error reading config file:", err, "Using default configuration")
			return common.MakeConverterConfig()
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
		return common.MakeConverterConfig()
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

func readSong(filepath string, relpath string) common.Song {
	song := common.MakeSong()
	// return song
	tags, err := taglib.ReadTags(filepath)
	if err != nil {
		log.Fatal(err)
		return song
	}

	if len(tags[taglib.Album]) > 0 {
		song.Album = tags[taglib.Album][0]
	}
	
	if len(tags[taglib.Artist]) > 0 {
		if len(tags[taglib.Artist]) > 1 {
			song.Artist = strings.Join(tags[taglib.Artist], ", ")
		} else {
			song.Artist = tags[taglib.Artist][0]
		}
	}

	if len(tags[taglib.AlbumArtist]) > 0 {
		song.AlbumArtist = tags[taglib.AlbumArtist][0]
	}

	if len(tags[taglib.Title]) > 0 {
		song.Title = tags[taglib.Title][0]
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

func addSongsRecursive(dir string, reldir string, songs map[string]common.Song) {
	fileSystem := os.DirFS(dir)
	fs.WalkDir(fileSystem, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.Type().IsDir() {	
			ext := getFileExtension(dirEntry.Name())
			if ext != dirEntry.Name() && slices.Contains(WhitelistedFiletypes, strings.ToUpper(ext)) {
				key := reldir + "/" + path
				filepath := osPathJoin(dir, path)
				
				// Only read song metadata if it has not already been loaded from db file
				if _, exists := songs[key]; !exists {
					songs[key] = readSong(filepath, key)
				}
			}
		}
		return nil
	})
}

func songMatch(config *common.ConverterConfig, song *common.Song, songKey string) bool {
	if config.FormatMatchType == common.Exact {
		isValid := true
		splitFormat := strings.Split(config.Format, "/")
		splitKey := strings.Split(songKey, "/")

		for i, field := range splitFormat {
			if field == common.ArtistFormat {
				isValid = isValid && (song.Artist == splitKey[i])
			} else if field == common.AlbumArtistFormat {
				isValid = isValid && (song.AlbumArtist == splitKey[i])
			} else if field == common.AlbumFormat {
				isValid = isValid && (song.Album == splitKey[i])
			} else if field == common.TitleFormat {
				isValid = isValid && (song.Title == splitKey[i])
			} else if field == common.TrackNumberFormat {
				keyInt, err := strconv.Atoi(splitKey[i])
				if err != nil {
					fmt.Println("ERROR: Format key not convertable to int")
				}

				isValid = isValid && (song.TrackNumber == keyInt)
			}
		}

		return isValid
	} else if config.FormatMatchType == common.RegularMatch {
		// TODO
	} else if  config.FormatMatchType == common.ReverseMatch {
		// TODO
	}

	return false
}


func matchSongsInList(config *common.ConverterConfig, list []string, songs map[string]common.Song) []*common.Song {
	songList := make([]*common.Song, len(list))

	// *Very* naive and inefficient implementation, maybe TODO streamline
	for _, song := range songs {
		for i, val := range list {
			if songMatch(config, &song, val) {
				songList[i] = &song
			}
		}
	}

	// fmt.Println("Got matches:", songList)
	return songList
}

func main() {
	kong.Parse(&CLI, kong.Description("A utility that takes in a CSV playlist of song metadata and converts it to a relative-pathed M3U playlist."))
	var config common.ConverterConfig
	foundSongs := make(map[string]common.Song)

	if (CLI.Config != "") {
		config = parseConfig(CLI.Config)
	} else {
		config = common.MakeConverterConfig()
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

	if _, err := os.Stat(ConverterDbFile); err == nil {
		fmt.Println("Existing db file found. Reading...")	
		gobFile, err := os.Open(ConverterDbFile)
		if err != nil {
			panic(err)
		}

		decoder := gob.NewDecoder(gobFile)
		decoder.Decode(&foundSongs)
		gobFile.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}
	
	fmt.Println("Building database...")
	for _, path := range config.Paths {
		fmt.Println("Reading", path)
		addSongsRecursive(path, filepath.Base(path), foundSongs)
	}

	fmt.Println("Writing database...")
	gobFile, err := os.Create(ConverterDbFile)
	if err != nil {
		panic(err)
	}
	encoder := gob.NewEncoder(gobFile)
	encoder.Encode(foundSongs)
	gobFile.Close()

	fmt.Println("Reading input playlist...")

	var inputType string
	if CLI.InputType != "" {
		inputType = strings.ToUpper(CLI.InputType)
	} else {
		inputType = strings.ToUpper(getFileExtension(CLI.Input))
	}

	var reader readers.PlaylistReader
	if slices.Contains(InputTypes, inputType) {
		if inputType == "CSV" {
			reader = readers.ReadCsv(CLI.Input, "default")
		} else if inputType == "EXPORTIFY" {
			reader = readers.ReadCsv(CLI.Input, "exportify")
		}
	} else {
		panic(fmt.Sprintln("Invalid reader type", inputType))
	}

	keyList := reader.GetKeyList(config.Format)
	if len(keyList) < 1 {
		panic("Keylist empty from reader")
	}

	var outputType string
	if CLI.OutputType != "" {
		outputType = strings.ToUpper(CLI.OutputType)
	} else {
		outputType = strings.ToUpper(getFileExtension(CLI.Output))
	}

	fmt.Println("Matching playlist items...")
	songList := matchSongsInList(&config, keyList, foundSongs)

	if CLI.Verbosity > 1 {
		fmt.Println("Couldn't find:")
		for i, song := range songList {
			if song == nil {
				fmt.Println(keyList[i])
			}
		}
	}

	fmt.Println("Writing output playlist...")

	if slices.Contains(OutputTypes, outputType) {
		if outputType == "M3U" {
			writers.WriteM3U(CLI.Output, songList)
		}
	} else {
		panic(fmt.Sprintln("Invalid reader type", inputType))
	}
}
