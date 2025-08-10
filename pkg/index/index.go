package index

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/jessegeens/go-toolbox/pkg/fs"
	//"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

type Index struct {
	Version int
	Entries []*Entry
}

func New(entries []*Entry) *Index {
	return &Index{
		Version: 2,
		Entries: entries,
	}
}

func Read(repo *repository.Repository) (*Index, error) {
	indexFile, err := repo.RepositoryFile(false, "index")
	if err != nil {
		return nil, err
	}

	// New repositories don't have an index file yet
	if !fs.Exists(indexFile) {
		return New([]*Entry{}), nil
	}

	index, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	return parseIndex(index)

}

func (i *Index) Write(repo *repository.Repository) error {
	filepath, err := repo.RepositoryFile(false, "index")
	if err != nil {
		return err
	}

	data := []byte{}

	// Write magic bytes
	data = append(data, []byte("DIRC")...)

	// Write version number
	data = writeUintToBytes(uint32(i.Version), data)

	// Write number of entries
	data = writeUintToBytes(uint32(len(i.Entries)), data)

	// Write the entries
	idx := 0
	for _, e := range i.Entries {
		// Write ctime
		ctimeUnix := uint(e.CTime.Unix())
		data = writeUintToBytes(uint64(ctimeUnix), data)

		// Write mtime
		mtimeUnix := uint(e.MTime.Unix())
		data = writeUintToBytes(uint64(mtimeUnix), data)

		// Device + inode
		data = writeUintToBytes(e.Dev, data)
		data = writeUintToBytes(e.Inode, data)

		// Next two bytes are unused, so we write two nullbytes
		data = append(data, []byte{0x00, 0x00}...)

		// Mode
		mode := (e.ModeType << 12) | ModeType(e.ModePerms)
		data = writeUintToBytes(uint16(mode), data)

		// UID, GID, Size
		data = writeUintToBytes(e.UID, data)
		data = writeUintToBytes(e.GID, data)
		data = writeUintToBytes(e.Size, data)

		// SHA
		sha, err := hex.DecodeString(e.SHA)
		if err != nil {
			return fmt.Errorf("failed to write index: invalid sha: %s", err.Error())
		}
		if len(sha) != 20 {
			return fmt.Errorf("failed to write index: invalid sha length: %d", len(sha))
		}
		data = append(data, sha...)

		// Name length and flags
		flagAsssumeValid := uint16(0)
		if e.FlagAssumeValid {
			flagAsssumeValid = 0x1 << 15
		}
		// 0011 0000 0000 0000 = 12288
		flagStage := e.FlagStage & uint16(12288)
		nameLen := min(len(e.Name), 0xFF)
		nameFlags := flagAsssumeValid | flagStage | uint16(nameLen)
		data = writeUintToBytes(nameFlags, data)

		// Name
		data = append(data, []byte(e.Name)...)
		data = append(data, 0x0)

		// Padding
		idx = 62 + len(e.Name) + 1
		if idx%8 != 0 {
			for range 8 - (idx % 8) {
				data = append(data, 0x0)
			}
		}
	}

	os.WriteFile(filepath, data, os.ModePerm)
	return nil
}

func parseIndex(index []byte) (*Index, error) {
	// Implement length check
	if len(index) < 12 {
		return nil, errors.New("invalid index: too short")
	}

	enc := binary.BigEndian
	entries := []*Entry{}

	header := index[:12]
	signature := header[:4]
	if !bytes.Equal(signature, []byte("DIRC")) {
		return nil, errors.New("invalid index signature")
	}

	version := enc.Uint32(header[4:8])
	if version != 2 {
		return nil, errors.New("invalid index version: got only supports git index version 2; got " + strconv.Itoa(int(version)))
	}

	count := enc.Uint32(header[8:12])
	content := index[12:]
	idx := 0

	for range count {
		if len(content) < idx+62 {
			break
		}
		// TODO: bounds check on content
		entry := &Entry{}

		// Read creation time seconds as unix timestamp
		ctimeSec := enc.Uint32(content[idx : idx+4])
		// Read creation time nanoseconds
		ctimeNano := enc.Uint32(content[idx+4 : idx+8])
		entry.CTime = time.Unix(int64(ctimeSec), int64(ctimeNano))

		// Same for modification time
		mtimeSec := enc.Uint32(content[idx+8 : idx+12])
		// Read creation time nanoseconds
		mtimeNano := enc.Uint32(content[idx+12 : idx+16])
		entry.MTime = time.Unix(int64(mtimeSec), int64(mtimeNano))

		// Read device id and inode
		entry.Dev = enc.Uint32(content[idx+16 : idx+20])
		entry.Inode = enc.Uint32(content[idx+20 : idx+24])

		// The next two bytes are unused, so we ignore them

		// Read mode type and permissions
		mode := enc.Uint16(content[idx+26 : idx+28])
		modeType := mode >> 12
		if !isValidModeType(modeType) {
			return nil, fmt.Errorf("invalid mode type: %d", modeType)
		}
		modePerms := mode & uint16(511) // 0000000111111111
		entry.ModeType = ModeType(modeType)
		entry.ModePerms = modePerms

		entry.UID = enc.Uint32(content[idx+28 : idx+32])
		entry.GID = enc.Uint32(content[idx+32 : idx+36])
		entry.Size = enc.Uint32(content[idx+36 : idx+40])

		// Parse SHA
		sha := content[idx+40 : idx+60]
		entry.SHA = hex.EncodeToString(sha)

		// Parse flags
		flags := enc.Uint16(content[idx+60 : idx+62])
		// 1000 0000 0000 0000 = 32768
		entry.FlagAssumeValid = (flags & uint16(32768)) != 0
		// 0100 0000 0000 0000 = 16384
		extended := (flags & uint16(16384)) != 0
		if extended {
			return nil, errors.New("extended mode not supported")
		}
		// 0011 0000 0000 0000 = 12288
		entry.FlagStage = (flags & uint16(12288))
		// 0000 1111 1111 1111 = 4095
		nameLength := flags & uint16(4095)

		// Now we've read 62 bytes, so we advance the index
		idx += 62

		// We read the name
		if nameLength < 0xFF {
			if len(content) < idx+int(nameLength) || content[idx+int(nameLength)] != 0 {
				return nil, errors.New("invalid name length in index")
			}
			entry.Name = string(content[idx : idx+int(nameLength)])
			idx += int(nameLength) + 1 // Extra byte for the null byte at the end
		} else {
			// If the name is too long, we find the first occurence as a null byte as the demarcator
			len := findNullByteIndex(content[idx:])
			if len < 0 {
				return nil, errors.New("invalid name in index")
			}
			entry.Name = string(content[idx : idx+len])
			idx += len + 1
		}

		// index must be multiple of eight, since data is padded for ptr alignment
		idx = 8 * int(math.Ceil(float64(idx)/8))

		entries = append(entries, entry)
	}

	return New(entries), nil
}

func findNullByteIndex(arr []byte) int {
	for i, v := range arr {
		if v == byte(0) {
			return i
		}
	}

	return -1
}

func writeUintToBytes[I uint16 | uint32 | uint64](num I, data []byte) []byte {
	enc := binary.BigEndian

	switch any(num).(type) {
	case uint16:
		temp := make([]byte, 2)
		enc.PutUint16(temp, uint16(num))
		data = append(data, temp...)
	case uint32:
		temp := make([]byte, 4)
		enc.PutUint32(temp, uint32(num))
		data = append(data, temp...)
	case uint64:
		temp := make([]byte, 8)
		enc.PutUint64(temp, uint64(num))
		data = append(data, temp...)
	}

	return data
}
