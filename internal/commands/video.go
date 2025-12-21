package commands

import "github.com/urfave/cli/v2"

// VideoCommands configures the CLI subcommands for working with indexed videos.
var VideoCommands = &cli.Command{
	Name:  "video",
	Usage: "Video subcommands",
	Subcommands: []*cli.Command{
		VideoListCommand,
		VideoTrimCommand,
		VideoRemuxCommand,
		VideoTranscodeCommand,
		VideoInfoCommand,
	},
}

// videoCountFlag limits the number of results returned by video commands.
var videoCountFlag = &cli.UintFlag{
	Name:    "count",
	Aliases: []string{"n"},
	Usage:   "maximum `NUMBER` of results",
	Value:   10000,
}

// videoIncludeSidecarFlag includes sidecar video files in list output.
var videoIncludeSidecarFlag = &cli.BoolFlag{
	Name:  "include-sidecar",
	Usage: "include sidecar video files in results",
}

// videoForceFlag allows overwriting existing output files for remux/transcode.
var videoForceFlag = &cli.BoolFlag{
	Name:    "force",
	Aliases: []string{"f"},
	Usage:   "replace existing output files",
}

// videoNoBackupFlag skips creating .backup files for in-place mutations.
var videoNoBackupFlag = &cli.BoolFlag{
	Name:  "no-backup",
	Usage: "do not keep a .backup copy of original files",
}

// videoVerboseFlag adds raw metadata to video info output.
var videoVerboseFlag = &cli.BoolFlag{
	Name:  "verbose",
	Usage: "include raw metadata output",
}
