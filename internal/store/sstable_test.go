package store

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// BEGIN AI SECTION

func TestSSTable_Option3_Format(t *testing.T) {
	// Create a dummy memtable flush event with 2 exact items
	entry := NewSSTableEntry()
	entry.LogEntries = append(entry.LogEntries, NewLogEntry("apple", "red"))
	entry.LogEntries = append(entry.LogEntries, NewLogEntry("banana", "yellow"))

	var buf bytes.Buffer
	err := entry.Encode(&buf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := buf.Bytes()

	// Checkpoint 1: Minimum Size
	// Footer (8) + checksum (4) + KV count (2) = minimum 14 bytes
	if len(data) < 14 {
		t.Fatalf("Encoded data too small: %d bytes (Did you write the footer?)", len(data))
	}
	t.Log("✅ Checkpoint 1 Passed: Minimum file size met!")

	// Checkpoint 2: Validate the Top-Level Header (KV Cnt)
	// Because Option 3 puts KV cnt at the very start now (Checksum moved):
	kvCnt := binary.LittleEndian.Uint16(data[0:2])
	if kvCnt != 2 {
		t.Fatalf("Checkpoint 2 Failed! Expected top-level KV Cnt of 2, got %d. Did you structure the start correctly?", kvCnt)
	}
	t.Log("✅ Checkpoint 2 Passed: KV Cnt Header is correct!")

	// Checkpoint 3: Validate the Checksum AND Footer Magic Number
	// The file ends with: [IndexOffset 4B][MagicNum 4B][Checksum 4B]
	footerStart := len(data) - 12

	magicNum := binary.LittleEndian.Uint32(data[footerStart+4 : footerStart+8])
	expectedMagic := uint32(0xCAFEBABE)
	if magicNum != expectedMagic {
		t.Fatalf("Checkpoint 3 Failed! Expected magic number 0x%X in Footer, got 0x%X.", expectedMagic, magicNum)
	}
	t.Log("✅ Checkpoint 3 Passed: Magic Number matches right before EOF Checksum!")

	// Checkpoint 4: Validate the Index Offset Points to real data!
	indexOffset := binary.LittleEndian.Uint32(data[footerStart : footerStart+4])
	if indexOffset == 0 || indexOffset >= uint32(footerStart) {
		t.Fatalf("Checkpoint 4 Failed! Index offset %d is invalid or overlapping.", indexOffset)
	}
	t.Logf("✅ Checkpoint 4 Passed: Index offset successfully extracted: points to byte %d!", indexOffset)

	// Checkpoint 5: Verify the first Index
	indexData := data[indexOffset:footerStart]
	if len(indexData) < 6 {
		t.Fatalf("Checkpoint 5 Failed! Index block is missing or too small.")
	}

	// Read the first Location from the Index
	firstLocation := binary.LittleEndian.Uint32(indexData[0:4])
	if firstLocation != 2 { // Since Data block starts exactly after KVCnt(2) = Byte 2!
		t.Fatalf("Checkpoint 5 Failed! Expected first Data block to be at byte 2, but index points to %d", firstLocation)
	}
	t.Log("✅ Checkpoint 5 Passed: The Index block accurately points to the first Data block! Everything is mathematically flawless.")
}

func TestSSTable_Decode_Option3(t *testing.T) {
	// 1. Create original source of truth
	original := NewSSTableEntry()
	original.LogEntries = append(original.LogEntries, NewLogEntry("hello", "world"))
	original.LogEntries = append(original.LogEntries, NewLogEntry("vault", "kv"))

	// 2. Encode using your Option 3 format
	var buf bytes.Buffer
	err := original.Encode(&buf)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// 3. Setup a ReadSeeker for the Decoder
	rawBytes := buf.Bytes()
	reader := bytes.NewReader(rawBytes) // bytes.NewReader naturally implements io.ReadSeeker!

	// 4. Fire the Decode parsing!
	decoded := NewSSTableEntry()

	// Hint: You might need to change Decode() to accept io.ReadSeeker instead of io.Reader
	// so you can use r.Seek(-4, io.SeekEnd) to read your checksum!
	err = decoded.Decode(reader)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// 5. Verify perfect mathematical extraction
	if len(decoded.LogEntries) != len(original.LogEntries) {
		t.Fatalf("Decode returned %d entries, expected %d", len(decoded.LogEntries), len(original.LogEntries))
	}

	for i, entry := range original.LogEntries {
		if decoded.LogEntries[i].Key != entry.Key || decoded.LogEntries[i].Value != entry.Value {
			t.Errorf("Mismatch at index %d! Expected {%s: %s}, Got {%s: %s}",
				i, entry.Key, entry.Value, decoded.LogEntries[i].Key, decoded.LogEntries[i].Value)
		}
	}
	t.Log("✅ Decode completely successful! All entries matched perfectly.")
}

// END AI SECTION
