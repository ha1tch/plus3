// file: pkg/diskimg/hostio_test.go

package diskimg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func createTestFile(t *testing.T, content []byte) string {
	tmpFile, err := os.CreateTemp("", "plus3test")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func TestImportFile(t *testing.T) {
	disk := NewDiskImage()

	// Create test file
	testData := []byte("10 PRINT \"HELLO\"\n20 GOTO 10\n")
	hostPath := createTestFile(t, testData)
	defer os.Remove(hostPath)

	// Test basic import
	t.Run("Basic import", func(t *testing.T) {
		err := disk.ImportFile(hostPath, "TEST.BAS", nil)
		if err != nil {
			t.Fatalf("Import failed: %v", err)
		}

		// Verify file
		f, err := disk.OpenFile("TEST.BAS", false)
		if err != nil {
			t.Fatalf("Failed to open imported file: %v", err)
		}
		defer f.Close()

		data := make([]byte, len(testData))
		n, err := f.Read(data)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if n != len(testData) {
			t.Errorf("Wrong read size: got %d, want %d", n, len(testData))
		}
		if !bytes.Equal(data, testData) {
			t.Error("File content mismatch")
		}
	})

	// Test import with header
	t.Run("Import with header", func(t *testing.T) {
		opts := &ImportOptions{
			AddHeader: true,
			FileType:  FileTypeProgram,
			Line:      10,
		}

		err := disk.ImportFile(hostPath, "TEST2.BAS", opts)
		if err != nil {
			t.Fatalf("Import with header failed: %v", err)
		}

		// Verify file has header
		f, err := disk.OpenFile("TEST2.BAS", false)
		if err != nil {
			t.Fatalf("Failed to open imported file: %v", err)
		}
		defer f.Close()

		if !f.isHeadered {
			t.Error("File should have header")
		}

		// Read and verify header
		headerData := make([]byte, HeaderSize)
		_, err = f.ReadAt(headerData, 0)
		if err != nil {
			t.Fatalf("Failed to read header: %v", err)
		}

		header := &Plus3DosHeader{}
		err = header.FromBytes(headerData)
		if err != nil {
			t.Fatalf("Failed to parse header: %v", err)
		}

		if header.FileLength != uint32(len(testData)) {
			t.Errorf("Wrong file length in header: got %d, want %d",
				header.FileLength, len(testData))
		}
	})
}

func TestImportBasicProgram(t *testing.T) {
	disk := NewDiskImage()

	// Create test BASIC program
	program := []byte("10 PRINT \"HELLO\"\n20 GOTO 10\n")
	hostPath := createTestFile(t, program)
	defer os.Remove(hostPath)

	err := disk.ImportBasicProgram(hostPath, 10)
	if err != nil {
		t.Fatalf("ImportBasicProgram failed: %v", err)
	}

	// Get imported filename
	base := filepath.Base(hostPath)
	diskName := base[:len(base)-len(filepath.Ext(base))] + ".BAS"

	// Verify file
	f, err := disk.OpenFile(diskName, false)
	if err != nil {
		t.Fatalf("Failed to open imported file: %v", err)
	}
	defer f.Close()

	if !f.isHeadered {
		t.Error("BASIC program should have header")
	}

	fileType, _, line, _ := f.header.GetBasicHeader()
	if fileType != FileTypeProgram {
		t.Error("Wrong file type in header")
	}
	if line != 10 {
		t.Errorf("Wrong LINE parameter: got %d, want 10", line)
	}
}

func TestImportScreen(t *testing.T) {
	disk := NewDiskImage()

	// Create test screen file
	screen := make([]byte, 6912)
	for i := range screen {
		screen[i] = byte(i & 0xFF)
	}
	hostPath := createTestFile(t, screen)
	defer os.Remove(hostPath)

	err := disk.ImportScreen(hostPath)
	if err != nil {
		t.Fatalf("ImportScreen failed: %v", err)
	}

	// Get imported filename
	base := filepath.Base(hostPath)
	diskName := base[:len(base)-len(filepath.Ext(base))] + ".SCR"

	// Verify file
	f, err := disk.OpenFile(diskName, false)
	if err != nil {
		t.Fatalf("Failed to open imported file: %v", err)
	}
	defer f.Close()

	if !f.isHeadered {
		t.Error("Screen file should have header")
	}

	fileType, _, loadAddr, _ := f.header.GetBasicHeader()
	if fileType != FileTypeCode {
		t.Error("Wrong file type in header")
	}
	if loadAddr != 16384 {
		t.Errorf("Wrong load address: got %d, want 16384", loadAddr)
	}

	// Verify screen data
	data := make([]byte, 6912)
	_, err = f.Read(data)
	if err != nil {
		t.Fatalf("Failed to read screen data: %v", err)
	}

	if !bytes.Equal(data, screen) {
		t.Error("Screen data mismatch")
	}
}

func TestExportFile(t *testing.T) {
	disk := NewDiskImage()
	
	// Create and write test data
	testData := []byte("10 PRINT \"HELLO\"\n20 GOTO 10\n")
	f, _ := disk.OpenFile("TEST.BAS", true)
	header := NewPlus3DosHeader()
	header.SetBasicHeader(FileTypeProgram, uint16(len(testData)), 10, uint16(len(testData)))
	f.Write(header.toBytes())
	f.Write(testData)
	f.Close()

	tmpDir, err := os.MkdirTemp("", "plus3test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Export with header", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "with_header.bas")
		err := disk.ExportFile("TEST.BAS", outPath, false)
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != HeaderSize+len(testData) {
			t.Error("Exported file wrong size")
		}
	})

	t.Run("Export without header", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "no_header.bas")
		err := disk.ExportFile("TEST.BAS", outPath, true)
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, testData) {
			t.Error("Exported data mismatch")
		}
	})

	t.Run("Extract BASIC", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "program.bas")
		err := disk.ExtractBasic("TEST.BAS", outPath)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, testData) {
			t.Error("Extracted BASIC data mismatch")
		}
	})
}

func TestImportErrors(t *testing.T) {
	disk := NewDiskImage()

	t.Run("Non-existent file", func(t *testing.T) {
		err := disk.ImportFile("nonexistent.txt", "test.txt", nil)
		if err == nil {
			t.Error("Import should fail for non-existent file")
		}
	})

	t.Run("Invalid screen size", func(t *testing.T) {
		invalidScreen := make([]byte, 1000)
		path := createTestFile(t, invalidScreen)
		defer os.Remove(path)

		err := disk.ImportScreen(path)
		if err == nil {
			t.Error("ImportScreen should fail for wrong size")
		}
	})

	t.Run("Large file", func(t *testing.T) {
		largeFile := make([]byte, 9*1024*1024) // 9MB
		path := createTestFile(t, largeFile)
		defer os.Remove(path)

		err := disk.ImportFile(path, "large.dat", nil)
		if err == nil {
			t.Error("Import should fail for files > 8MB")
		}
	})
}