// file: pkg/diskimg/all_test.go

package diskimg

import (
	"bytes"
	"io"
	"testing"
)

func TestCompleteFileOperations(t *testing.T) {
	disk := NewDiskImage()

	// Test 1: Create basic file with header
	t.Run("Create and write BASIC program", func(t *testing.T) {
		f, err := disk.OpenFile("PROGRAM.BAS", true)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		header := NewPlus3DosHeader()
		err = header.SetBasicHeader(FileTypeProgram, 100, 10, 90)
		if err != nil {
			t.Fatalf("Failed to set header: %v", err)
		}

		_, err = f.Write(header.toBytes())
		if err != nil {
			t.Fatalf("Failed to write header: %v", err)
		}

		program := []byte("10 PRINT \"HELLO\"\n20 GOTO 10\n")
		_, err = f.Write(program)
		if err != nil {
			t.Fatalf("Failed to write program: %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Failed to close file: %v", err)
		}
	})

	// Test 2: Create CODE file with attributes
	t.Run("Create CODE file with attributes", func(t *testing.T) {
		f, err := disk.OpenFile("SCREEN.SCR", true)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		header := NewPlus3DosHeader()
		err = header.SetBasicHeader(FileTypeCode, 6912, 16384, 0) // Standard screen$ size and address
		if err != nil {
			t.Fatalf("Failed to set header: %v", err)
		}

		_, err = f.Write(header.toBytes())
		if err != nil {
			t.Fatalf("Failed to write header: %v", err)
		}

		screenData := make([]byte, 6912)
		for i := range screenData {
			screenData[i] = byte(i & 0xFF) // Test pattern
		}
		
		_, err = f.Write(screenData)
		if err != nil {
			t.Fatalf("Failed to write screen data: %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Failed to close file: %v", err)
		}

		// Set file attributes
		entry, _, err := disk.directory.FindFile("SCREEN.SCR")
		if err != nil {
			t.Fatalf("Failed to find file: %v", err)
		}

		attrs := &FileAttributes{ReadOnly: true, System: true}
		attrs.ApplyToDirectoryEntry(entry)
	})

	// Test 3: Large file handling and fragmentation
	t.Run("Large file handling", func(t *testing.T) {
		f, err := disk.OpenFile("LARGE.DAT", true)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Write data larger than one block
		data := make([]byte, BlockSize*3+512)
		for i := range data {
			data[i] = byte(i & 0xFF)
		}

		_, err = f.Write(data)
		if err != nil {
			t.Fatalf("Failed to write large file: %v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("Failed to close file: %v", err)
		}

		// Verify data
		f, err = disk.OpenFile("LARGE.DAT", false)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		readData := make([]byte, len(data))
		_, err = io.ReadFull(f, readData)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !bytes.Equal(data, readData) {
			t.Error("Read data doesn't match written data")
		}
	})

	// Test 4: Directory operations and file deletion
	t.Run("Directory operations", func(t *testing.T) {
		// Check directory entries
		files := []string{"PROGRAM.BAS", "SCREEN.SCR", "LARGE.DAT"}
		for _, filename := range files {
			entry, _, err := disk.directory.FindFile(filename)
			if err != nil {
				t.Errorf("Failed to find %s: %v", filename, err)
				continue
			}

			if entry.IsDeleted() || entry.IsUnused() {
				t.Errorf("File %s should exist", filename)
			}
		}

		// Delete a file
		err := disk.directory.DeleteFile("PROGRAM.BAS")
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// Verify deletion
		_, _, err = disk.directory.FindFile("PROGRAM.BAS")
		if err == nil {
			t.Error("Deleted file still exists")
		}
	})

	// Test 5: Error conditions
	t.Run("Error handling", func(t *testing.T) {
		// Try to open non-existent file
		_, err := disk.OpenFile("NOTFOUND.DAT", false)
		if err == nil {
			t.Error("Opening non-existent file should fail")
		}

		// Try to create duplicate file
		_, err = disk.OpenFile("SCREEN.SCR", true)
		if err == nil {
			t.Error("Creating duplicate file should fail")
		}

		// Try to write to read-only file
		f, err := disk.OpenFile("SCREEN.SCR", false)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		_, err = f.Write([]byte("test"))
		if err == nil {
			t.Error("Writing to read-only file should fail")
		}
	})
}