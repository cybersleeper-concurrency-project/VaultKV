package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

type RecordType uint8

const (
	RecordTypePut RecordType = iota
	RecordTypeDelete
)

type LogEntry struct {
	Type  RecordType
	Key   string
	Value string
}

type WAL struct {
	file *os.File
	mu   sync.Mutex
}

func NewWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		file: file,
		mu:   sync.Mutex{},
	}
	return wal, nil
}

func (w *WAL) Append(entry *LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := entry.Encode(w.file); err != nil {
		return err
	}
	if err := w.file.Sync(); err != nil {
		return err
	}
	return nil
}

func (w *WAL) ReadAll() ([]*LogEntry, error) {
	w.file.Seek(0, io.SeekStart)

	var entries []*LogEntry

	for {
		entry := &LogEntry{}
		err := entry.Decode(w.file)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func BatchBinaryWrite(w io.Writer, values ...any) error {
	for _, v := range values {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}

func BatchBinaryRead(r io.Reader, values ...any) error {
	for _, v := range values {
		if err := binary.Read(r, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}

func (e *LogEntry) Encode(w io.Writer) error {
	keyLen := uint16(len(e.Key))
	valLen := uint32(len(e.Value))
	var buf bytes.Buffer

	if err := BatchBinaryWrite(&buf, e.Type, keyLen, valLen); err != nil {
		return err
	}

	if _, err := buf.Write([]byte(e.Key)); err != nil {
		return err
	}
	if _, err := buf.Write([]byte(e.Value)); err != nil {
		return err
	}

	data := buf.Bytes()
	checksum := crc32.ChecksumIEEE(data)

	if err := binary.Write(w, binary.LittleEndian, checksum); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

func (e *LogEntry) Decode(r io.Reader) error {
	var checksum uint32
	var keyLen uint16
	var valLen uint32

	if err := BatchBinaryRead(r, &checksum, &e.Type, &keyLen, &valLen); err != nil {
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

	var buf bytes.Buffer
	BatchBinaryWrite(&buf, e.Type, keyLen, valLen, keyBytes, valBytes)

	if crc32.ChecksumIEEE(buf.Bytes()) != checksum {
		return fmt.Errorf("Corrupted log entry detected! Expected checksum: %d", checksum)
	}

	e.Key = string(keyBytes)
	e.Value = string(valBytes)

	return nil
}
