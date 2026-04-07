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

	// Checkpoint 2: Validate the Top-Level Header (Checksum + KV Cnt)
	// Because Option 3 puts KV cnt right after the Checksum:
	kvCnt := binary.LittleEndian.Uint16(data[4:6])
	if kvCnt != 2 {
		t.Fatalf("Checkpoint 2 Failed! Expected top-level KV Cnt of 2, got %d. Did you structure the start correctly?", kvCnt)
	}
	t.Log("✅ Checkpoint 2 Passed: KV Cnt Header is correct!")

	// Checkpoint 3: Validate the Footer Magic Number
	// The Footer is the final 8 bytes -> [IndexOffset 4B][MagicNum 4B]
	footerStart := len(data) - 8
	magicNum := binary.LittleEndian.Uint32(data[footerStart+4:])

	expectedMagic := uint32(0xCAFEBABE) // Change this if you used a different hex number!
	if magicNum != expectedMagic {
		t.Fatalf("Checkpoint 3 Failed! Expected magic number 0x%X in Footer, got 0x%X.", expectedMagic, magicNum)
	}
	t.Log("✅ Checkpoint 3 Passed: Magic Number matches at EOF!")

	// Checkpoint 4: Validate the Index Offset Points to real data!
	indexOffset := binary.LittleEndian.Uint32(data[footerStart : footerStart+4])
	if indexOffset == 0 || indexOffset >= uint32(footerStart) {
		t.Fatalf("Checkpoint 4 Failed! Index offset %d is invalid or overlapping. footerStart: %d", indexOffset, footerStart)
	}
	t.Logf("✅ Checkpoint 4 Passed: Index offset successfully extracted: points to byte %d!", indexOffset)

	// Checkpoint 5: Verify the first Index
	// Jump to where the Footer told us the Index starts
	indexData := data[indexOffset:footerStart]
	if len(indexData) < 10 { // Location(8B) + KeyLen(2B) = 10 minimum
		t.Fatalf("Checkpoint 5 Failed! Index block is missing or too small.")
	}

	// Read the first Location from the Index
	firstLocation := binary.LittleEndian.Uint32(indexData[0:4])
	if firstLocation != 6 { // Since Data block start exactly after Checksum(4) + KVCnt(2) = Byte 6!
		t.Fatalf("Checkpoint 5 Failed! Expected first Data block to be at byte 6, but index points to %d", firstLocation)
	}
	t.Log("✅ Checkpoint 5 Passed: The Index block accurately points to the first Data block! Everything is mathematically flawless.")
}

// END AI SECTION
