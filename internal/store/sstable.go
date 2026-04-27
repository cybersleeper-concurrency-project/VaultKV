package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os"
)

const sstableLevelCount = 10
const sstableEntryLimit = 10

const maxEntryCntBytes = math.MaxUint16

const magicNumber uint32 = 0xCAFEBABE

type SSTable struct {
	file    *os.File
	Entries []*SSTableLevel
}

type SSTableEntry struct {
	LogEntries []*LogEntry
}

type SSTableLevel struct {
	file *os.File
}

type indexEntry struct {
	pointer uint32
	keyLen  uint16
	key     string
}

func NewSSTable(path string) (*SSTable, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	sstable := &SSTable{
		file:    file,
		Entries: make([]*SSTableLevel, 0),
	}
	return sstable, nil
}

func NewSSTableEntry() *SSTableEntry {
	return &SSTableEntry{
		LogEntries: make([]*LogEntry, 0),
	}
}

func (s *SSTable) MergeSkiplist(skiplist *Skiplist) (*SSTableEntry, error) {
	curNode := skiplist.BeginNode
	sstableEntry := NewSSTableEntry()

	if curNode == nil {
		return nil, fmt.Errorf("Skiplist is empty")
	}

	for {
		sstableEntry.LogEntries = append(sstableEntry.LogEntries, NewLogEntry(curNode.Key, curNode.Value))
		if curNode.Next[0] != nil {
			curNode = curNode.Next[0]
		} else {
			break
		}
	}

	return sstableEntry, nil
}

func (s *SSTable) Append(entry *SSTableEntry) error {
	if entry == nil {
		return fmt.Errorf("nil sst entry")
	}
	if s.file == nil {
		return fmt.Errorf("sst is nil")
	}

	if err := entry.Encode(s.file); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	return nil
}

func (s *SSTable) Flush(skiplist *Skiplist) error {
	sstEntry, err := s.MergeSkiplist(skiplist)
	if err != nil {
		return err
	}

	if err := s.Append(sstEntry); err != nil {
		return err
	}
	return nil
}

// Compaction need 1 argumen, that is the target SST Level.
// If the target lvl is 0, that means it will flush the
// Skiplist first
func (s *SSTable) Compaction(toLevel int) {
	// TODO

}

func (e *SSTableEntry) Encode(w io.Writer) error {
	if uint16(len(e.LogEntries)) > maxEntryCntBytes {
		return fmt.Errorf("entry length too large: %d, limit=%d", len(e.LogEntries), maxEntryCntBytes)
	}

	var dataLen uint16
	dataLen = uint16(len(e.LogEntries))

	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, dataLen); err != nil {
		return err
	}

	indexSlice := make([]indexEntry, dataLen)

	for i, v := range e.LogEntries {
		var keyLen uint16
		var valLen uint32

		keyLen = uint16(len(v.Key))
		valLen = uint32(len(v.Value))

		indexSlice[i] = indexEntry{
			pointer: uint32(buf.Len()),
			keyLen:  keyLen,
			key:     v.Key,
		}

		if err := binary.Write(&buf, binary.LittleEndian, keyLen); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, valLen); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(v.Key)); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(v.Value)); err != nil {
			return err
		}
	}

	// The Trailer Index Block starts from buf.Len()
	indexOffset := uint32(buf.Len())

	// Index Block
	for _, v := range indexSlice {
		if err := binary.Write(&buf, binary.LittleEndian, v.pointer); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, v.keyLen); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(v.key)); err != nil {
			return err
		}
	}

	// Write Footer directly into buf!
	if err := binary.Write(&buf, binary.LittleEndian, indexOffset); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.LittleEndian, magicNumber); err != nil {
		return err
	}

	data := buf.Bytes()

	// TODO(Future Improvement):
	// Currently, this writes a single Master Checksum over the entire SSTable file at the EOF
	// For production-grade durability against single-bit flips, consider rotating to
	// Block-Based Checksums (calculating and inserting a checksum every 4KB of data chunked)
	// to prevent a 1-byte corruption from invalidating the entire file
	checksum := crc32.ChecksumIEEE(data)

	// Write everything to disk, and Checksum at the absolute end
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, checksum); err != nil {
		return err
	}

	return nil
}

func (e *SSTableEntry) Decode(r io.Reader) error {
	var dataLen uint16

	var buf bytes.Buffer

	if err := binary.Read(r, binary.LittleEndian, &dataLen); err != nil {
		return err
	}

	if err := binary.Write(&buf, binary.LittleEndian, dataLen); err != nil {
		return err
	}

	logEntries := make([]*LogEntry, dataLen)

	for i := range dataLen {
		var keyLen uint16
		var valLen uint32

		if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &valLen); err != nil {
			return err
		}

		keyBytes := make([]byte, keyLen)
		valBytes := make([]byte, valLen)

		if _, err := io.ReadFull(r, keyBytes); err != nil {
			return err
		}
		if _, err := io.ReadFull(r, valBytes); err != nil {
			return err
		}

		// Wrtie to buf for the checksum later
		if err := binary.Write(&buf, binary.LittleEndian, keyLen); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, valLen); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(keyBytes)); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(valBytes)); err != nil {
			return err
		}

		logEntries[i] = &LogEntry{
			Key:   string(keyBytes),
			Value: string(valBytes),
		}
	}

	for range dataLen {
		var curIndex indexEntry

		if err := binary.Read(r, binary.LittleEndian, &curIndex.pointer); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &curIndex.keyLen); err != nil {
			return err
		}

		keyBytes := make([]byte, curIndex.keyLen)
		if _, err := io.ReadFull(r, keyBytes); err != nil {
			return err
		}

		if err := binary.Write(&buf, binary.LittleEndian, curIndex.pointer); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, curIndex.keyLen); err != nil {
			return err
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return err
		}
	}

	var curIndexOffset uint32
	var curMagicNumber uint32

	if err := binary.Read(r, binary.LittleEndian, &curIndexOffset); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &curMagicNumber); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.LittleEndian, curIndexOffset); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.LittleEndian, curMagicNumber); err != nil {
		return err
	}

	var checksum uint32
	if err := binary.Read(r, binary.LittleEndian, &checksum); err != nil {
		return err
	}

	if crc32.ChecksumIEEE(buf.Bytes()) != checksum {
		return fmt.Errorf("corrupted SST entry: expected checksum %d", checksum)
	}

	e.LogEntries = logEntries

	return nil
}

// TODO:
// - Determine how we gonna store the entries from LSM to SST (see WAL, it should be similar storing implementation)
// - Implement the MergeSSTableLevel
// - Implement the Compaction
