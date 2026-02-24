package store

import (
	"encoding/binary"
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
			if err == io.EOF {
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

	BatchBinaryWrite(w, e.Type, keyLen, valLen, e.Key, e.Value)

	return nil
}

func (e *LogEntry) Decode(r io.Reader) error {
	var keyLen uint16
	var valLen uint32

	BatchBinaryRead(r, &e.Type, &keyLen, &valLen)

	keyBytes := make([]byte, keyLen)
	valBytes := make([]byte, valLen)

	if _, err := io.ReadFull(r, keyBytes); err != nil {
		return err
	}
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return err
	}

	e.Key = string(keyBytes)
	e.Value = string(valBytes)

	return nil
}
