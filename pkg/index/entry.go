package index

import (
	"slices"
	"time"
)

type ModeType uint16

const (
	ModeTypeRegular = ModeType(uint16(8))  // 0000 1000
	ModeTypeSymlink = ModeType(uint16(10)) // 0000 1010
	ModeTypeGitlink = ModeType(uint16(14)) // 0000 1110
)

type Entry struct {
	// Creation Time
	CTime time.Time
	// Modification Time
	MTime time.Time
	// ID of Device containing this file
	Dev             uint32
	Inode           uint32
	ModeType        ModeType
	ModePerms       uint16
	UID             uint32
	GID             uint32
	Size            uint32
	SHA             string
	FlagAssumeValid bool
	FlagStage       uint16
	// Full path
	Name string
}

func isValidModeType(modeType uint16) bool {
	validModeTypes := []uint16{uint16(ModeTypeRegular), uint16(ModeTypeSymlink), uint16(ModeTypeGitlink)}
	return slices.Contains(validModeTypes, modeType)
}

func (m ModeType) String() string {
	switch m {
	case ModeTypeRegular:
		return "regular file"
	case ModeTypeSymlink:
		return "symlink"
	case ModeTypeGitlink:
		return "git link"
	default:
		return "invalid"
	}
}
