//go:build openbsd

package fsscript

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

func OpenFlagFromString(s string) (flag int, err error) {
	for part := range strings.SplitSeq(s, "|") {
		switch part {
		default:
			return flag, fmt.Errorf("unknown flag: %s", part)
		case "0":
			continue
		case "O_RDONLY":
			flag |= os.O_RDONLY
		case "O_WRONLY":
			flag |= os.O_WRONLY
		case "O_RDWR":
			flag |= os.O_RDWR
		case "O_APPEND":
			flag |= os.O_APPEND
		case "O_CREATE":
			flag |= os.O_CREATE
		case "O_EXCL":
			flag |= os.O_EXCL
		case "O_SYNC":
			flag |= os.O_SYNC
		case "O_TRUNC":
			flag |= os.O_TRUNC
		}
	}
	return
}

func FileTypeFromString(s string) (mode fs.FileMode, err error) {
	switch s {
	case "dir":
		return fs.ModeDir, nil
	case "file":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown file type: %s", s)
	}
}
