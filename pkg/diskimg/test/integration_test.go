// file: pkg/diskimg/integration_test.go

package diskimg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFullDiskOperations(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.dsk")
	disk := NewDiskImage()

	files := []struct {
		name    string
		content []byte
		typ     byte
	}{
		{"HELLO.BAS", []byte("10 PRINT \"HELLO\"\n"), FileTypeProgram},
		{"DATA.BIN", []byte{0xF3, 0xAF, 0x32}, FileTypeCode},
		{"TEXT.TXT", []byte("Sample text file"), 0xFF},
	}

	// Create test files
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		os.WriteFile(path, f.content, 0644)
	}

	t.Run("disk lifecycle", func(t *testing.T) {
		// Save empty disk
		err := disk.SaveToFile(diskPath)
		if err != nil {
			t.Fatalf("Failed to save empty disk: %v", err)
		}

		// Import files
		for _, f := range files {
			var err error
			path := filepath.Join(tmpDir, f.name)
			
			switch f.typ {
			case FileTypeProgram:
				err = disk.ImportBasicProgram(path, 10)
			case FileTypeCode:
				err = disk.ImportCode(path, 32768)
			default:
				err = disk.ImportRaw(path)
			}
			if err != nil {
				t.Errorf("Failed to import %s: %v", f.name, err)
			}
		}

		// Save and reload disk
		err = disk.SaveToFile(diskPath)
		if err != nil {
			t.Fatalf("Failed to save populated disk: %v", err)
		}

		newDisk, err := LoadFromFile(diskPath)
		if err != nil {
			t.Fatalf("Failed to reload disk: %v", err)
		}

		// Export and verify files 
		for _, f := range files {
			outPath := filepath.Join(tmpDir, "out_"+f.name)
			var err error
			
			switch f.typ {
			case FileTypeProgram:
				err = newDisk.ExtractBasic(f.name, outPath)
			case FileTypeCode:
				err = newDisk.ExportFile(f.name, outPath, true)
			default:
				err = newDisk.ExportFile(f.name, outPath, false)
			}
			if err != nil {
				t.Errorf("Failed to export %s: %v", f.name, err)
				continue
			}

			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Errorf("Failed to read exported %s: %v", f.name, err)
				continue
			}

			if string(data) != string(f.content) {
				t.Errorf("Content mismatch for %s", f.name)
			}
		}

		// Test file deletion
		err = newDisk.directory.DeleteFile("TEXT.TXT")
		if err != nil {
			t.Errorf("Failed to delete file: %v", err)
		}

		// Verify deletion
		_, _, err = newDisk.directory.FindFile("TEXT.TXT")
		if err == nil {
			t.Error("File still exists after deletion")
		}

		// Test disk space recovery
		f, err := newDisk.OpenFile("NEWFILE.DAT", true)
		if err != nil {
			t.Error("Failed to create file in recovered space")
		} else {
			f.Close()
		}
	})

	t.Run("error conditions", func(t *testing.T) {
		// Test file size limit
		largeData := make([]byte, 9*1024*1024)
		largePath := filepath.Join(tmpDir, "large.bin")
		os.WriteFile(largePath, largeData, 0644)

		err := disk.ImportRaw(largePath)
		if err == nil {
			t.Error("Should reject files > 8MB")
		}

		// Test duplicate filename
		err = disk.ImportRaw(filepath.Join(tmpDir, "HELLO.BAS"))
		if err == nil {
			t.Error("Should reject duplicate filename")
		}

		// Test invalid filenames
		err = disk.ImportRaw(filepath.Join(tmpDir, "VeryLongFileName.txt"))
		if err == nil {
			t.Error("Should reject long filenames")
		}
	})
}