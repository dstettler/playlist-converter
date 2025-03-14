# P2M3U
This is a super simple utility that converts CSVs created by Exportify (or any CSV with metadata to match) to relative-pathed M3U playlists.
I split the reading/writing out a bit to allow for more input/output formats, but the gist of what this does is
Input format with *metadata* -> Output format with paths

If you need something that just directly maps path-based playlists into other formats, there are likely existing converters elsewhere.
If not and I run into the issue in the future, I'll probably write a Python script and link a gist here.

Important to emphasize that I created this mostly for personal use, but figured I would put up the code for others. 
If you have issues modding/working with it feel free to yell at me in an issue and I can clean up some parts, but it's a bit messy since it
was never a priority of mine, and I also treated this project as a way to learn a bit more Go.

Currently the matching system just uses pre-determined constants to decide match likelihood. These can be modified with a config file
using TOML syntax. 
For boosting files on an individual basis I currently only use filetype, so FLAC will be prioritized over MP3, etc.
You can change these values with the config file.

Ex:
```toml
Paths = ["Z:/Music/FLAC Library", "Z:/Music/iTunes/etc"]
Format = "Artist\u001EAlbum\u001ETitle"
MinimumMatchAllowance = 0.9
SplitCharacter = ','
SpecialCases = ["Artist, with, commas, in, their, name"]

[FiletypeBonuses]
FLAC = 0.3
MP3 = 0.4
```
## Building
If you have Go installed, it *should* install dependencies with `go build`, and this does not require any installation, so just run the generated executable!