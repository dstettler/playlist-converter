package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	common "dstet.me/p2m3u/common"
	readers "dstet.me/p2m3u/readers"
	writers "dstet.me/p2m3u/writers"
	"github.com/alecthomas/kong"
	"github.com/pelletier/go-toml/v2"
	"go.senan.xyz/taglib"
)

var WhitelistedFiletypes = []string{
	"OGG",
	"MP3",
	"M4A",
	"FLAC",
	"WAV",
	"AIFF",
}

var InputTypes = []string{
	"CSV",
	"EXPORTIFY",
}

var OutputTypes = []string{
	"M3U",
}

var CLI struct {
	Input         string   `arg:"" help:"Input playlist" type:"path"`
	Output        string   `arg:"" help:"Output file" type:"path"`
	SearchDirs    []string `arg:"" help:"Directories to search" type:"path" optional:""`
	Config        string   `short:"c" help:"Config file to use" type:"path"`
	DbFile        string   `help:"Custom db file" type:"path" optional:""`
	OutputMissing string   `help:"File to output missing songs" type:"path" optional:""`
	InputType     string   `short:"i" help:"Mode to parse input file" optional:""`
	OutputType    string   `short:"o" help:"Mode to write output file" optional:""`
	Verbosity     int      `type:"counter" short:"v" optional:"" help:"Verbosity counter" default:"0"`
}

func parseConfig(filepath string) common.ConverterConfig {
	if _, err := os.Stat(filepath); err == nil {
		config := common.MakeConverterConfig()
		if fileContents, fileErr := os.ReadFile(filepath); fileErr != nil {
			fmt.Println("ERROR: Error reading config file:", err, "Using default configuration")
			return config
		} else {
			if err := toml.Unmarshal(fileContents, &config); err != nil {
				panic(err)
			}

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

func addSongsRecursive(dir string, reldir string, lib *common.ConverterLibrary, config *common.ConverterConfig) {
	fileSystem := os.DirFS(dir)
	fs.WalkDir(fileSystem, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.Type().IsDir() {
			ext := common.GetFileExtension(dirEntry.Name())
			if ext != dirEntry.Name() && slices.Contains(WhitelistedFiletypes, strings.ToUpper(ext)) {
				key := reldir + "/" + path
				filepath := osPathJoin(dir, path)

				// Only read song metadata if it has not already been loaded from db file
				if searchedId := lib.GetId(key); searchedId == -1 {
					id := lib.GetNewId(key)
					newSong := readSong(filepath, key)
					lib.Songs[id] = &newSong

					for _, artist := range common.ArtistSplit(newSong.Artist, config) {
						artist = strings.TrimSpace(artist)

						if artist != common.UnknownArtist {
							lib.ArtistsIndex[artist] = append(lib.ArtistsIndex[artist], id)
						}
					}

					for _, artist := range common.ArtistSplit(newSong.AlbumArtist, config) {
						artist = strings.TrimSpace(artist)

						if artist != common.UnknownArtist {
							lib.AlbumArtistsIndex[artist] = append(lib.AlbumArtistsIndex[artist], id)
						}
					}

					lib.AlbumsIndex[newSong.Album] = append(lib.AlbumsIndex[newSong.Album], id)
					lib.TitlesIndex[newSong.Title] = append(lib.TitlesIndex[newSong.Title], id)
				}
			}
		}
		return nil
	})
}

func matchSongsInList(config *common.ConverterConfig, list []string, lib *common.ConverterLibrary) []*common.Song {
	songList := make([]*common.Song, len(list))

	// Very naive and inefficient implementation, maybe TODO streamline
	for i, val := range list {
		song := lib.GetSongFromFormatString(val, config)
		songList[i] = song
	}

	// fmt.Println("Got matches:", songList)
	return songList
}

func main() {
	kong.Parse(&CLI, kong.Description("A utility that takes in a playlist of song metadata and converts it to a relative-pathed playlist."))
	var config common.ConverterConfig
	library := common.MakeLibrary()

	if CLI.Config != "" {
		config = parseConfig(CLI.Config)
	} else {
		config = common.MakeConverterConfig()
	}

	if len(CLI.SearchDirs) > 0 && config.Paths == nil {
		config.Paths = CLI.SearchDirs
	} else if len(CLI.SearchDirs) > 0 {
		config.Paths = append(config.Paths, CLI.SearchDirs...)
	}

	if config.Paths == nil {
		fmt.Println("ERROR: No search paths specified! Unable to continue.")
		return
	}

	library.TryReadDbFile(CLI.DbFile)

	fmt.Println("Building database...")
	for _, path := range config.Paths {
		fmt.Println("Reading", path)
		addSongsRecursive(path, filepath.Base(path), &library, &config)
	}

	fmt.Println("Writing database...")
	library.WriteDbFile(CLI.DbFile)

	fmt.Println("Reading input playlist...")

	var inputType string
	if CLI.InputType != "" {
		inputType = strings.ToUpper(CLI.InputType)
	} else {
		inputType = strings.ToUpper(common.GetFileExtension(CLI.Input))
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
		outputType = strings.ToUpper(common.GetFileExtension(CLI.Output))
	}

	fmt.Println("Matching playlist items...")
	songList := matchSongsInList(&config, keyList, &library)

	if CLI.OutputMissing != "" {
		f, err := os.Create(CLI.OutputMissing)
		if err == nil {
			f.WriteString("Couldn't find:\n")
			for i, song := range songList {
				if song == nil {
					f.WriteString(keyList[i] + "\n")
				}
			}
		} else {
			fmt.Println("ERROR: Error printing to", CLI.OutputMissing)
		}

		f.Close()
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
