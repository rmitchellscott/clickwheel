package main

import (
	"fmt"
	"os"
	"strings"

	"clickwheel/internal/restore"
)

func main() {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "--restore-") {
		fmt.Fprintln(os.Stderr, "usage: clickwheel-helper --restore-partition|--restore-write-fw <args...>")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "--restore-partition":
		err = restore.RunPartitionSubcommand(os.Args[2:])
	case "--restore-write-fw":
		err = restore.RunWriteFirmwareSubcommand(os.Args[2:])
	default:
		err = fmt.Errorf("unknown restore subcommand: %s", os.Args[1])
	}
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
