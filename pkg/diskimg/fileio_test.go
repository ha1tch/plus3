// file: pkg/diskimg/fileio_test.go

package diskimg

import (
	"bytes"
	"io"
	"testing"
)

func TestFileReadWrite(t *testing.T) {
	disk := NewDiskImage()

	// Create test file
	f, err := disk.OpenFile("TEST.BAS", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write test data
	testData := []byte("Hello, Spectrum +3!")
	n, err := f.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Short write: got %d bytes, want %d", n, len(testData))
	}

	// Seek to start
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}

	// Read back data
	readData := make([]byte, len(testData))
	n, err = f.Read(readData)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Short read: got %d bytes, want %d", n, len(testData))
	}
	if !bytes.Equal(readData, testData) {
		t.Error("Read data doesn't match written data")
	}

	err = f.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestFileHeader(t *testing.T) {
	disk := NewDiskImage()

	// Create file with header
	f, err := disk.OpenFile("TEST.BAS", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create and write header
	header := NewPlus3DosHeader()
	err = header.SetBasicHeader(FileTypeProgram, 100, 10, 90)
	if err != nil {
		t.Fatalf("Failed to set header: %v", err)
	}

	headerData := header.toBytes()
	n, err := f.WriteAt(headerData, 0)
	if err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if n != HeaderSize {
		t.Errorf("Wrong header write size: got %d, want %d", n, HeaderSize)
	}

	// Write program data
	testData := []byte("10 PRINT \"HELLO\"\n20 GOTO 10")
	n, err = f.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = f.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen and verify
	f, err = disk.OpenFile("TEST.BAS", false)
	if err != nil {
		t.Fatalf("Failed to reopen file: %v", err)
	}

	if !f.isHeadered {
		t.Error("File should be detected as having header")
	}

	// Verify header
	if f.header.FileLength != uint32(HeaderSize+len(testData)) {
		t.Errorf("Wrong file length in header: got %d, want %d",
			f.header.FileLength, HeaderSize+len(testData))
	}

	readData := make([]byte, len(testData))
	n, err = f.Read(readData)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}
	if !bytes.Equal(readData, testData) {
		t.Error("Read data doesn't match written data")
	}
}

func TestRandomAccess(t *testing.T) {
	disk := NewDiskImage()
	f, err := disk.OpenFile("TEST.DAT", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write data at different positions
	writes := []struct {
		pos  int64
		data []byte
	}{
		{0, []byte("First")},
		{100, []byte("Middle")},
		{1000, []byte("Last")},
	}

	for _, w := range writes {
		n, err := f.WriteAt(w.data, w.pos)
		if err != nil {
			t.Errorf("WriteAt(%d) failed: %v", w.pos, err)
		}
		if n != len(w.data) {
			t.Errorf("Short write at %d: got %d bytes, want %d",
				w.pos, n, len(w.data))
		}
	}

	// Verify data
	for _, w := range writes {
		data := make([]byte, len(w.data))
		n, err := f.ReadAt(data, w.pos)
		if err != nil {
			t.Errorf("ReadAt(%d) failed: %v", w.pos, err)
		}
		if n != len(w.data) {
			t.Errorf("Short read at %d: got %d bytes, want %d",
				w.pos, n, len(w.data))
		}
		if !bytes.Equal(data, w.data) {
			t.Errorf("Data mismatch at %d", w.pos)
		}
	}
}

func TestSeek(t *testing.T) {
	disk := NewDiskImage()
	f, err := disk.OpenFile("TEST.DAT", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write some data
	data := []byte("0123456789")
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	tests := []struct {
		offset    int64
		whence    int
		wantPos   int64
		wantError bool
	}{
		{5, io.SeekStart, 5, false},
		{2, io.SeekCurrent, 7, false},
		{-3, io.SeekEnd, 7, false},
		{0, io.SeekStart, 0, false},
		{-1, io.SeekStart, 0, true},
		{100, 99, 0, true}, // Invalid whence
	}

	for _, tt := range tests {
		pos, err := f.Seek(tt.offset, tt.whence)
		if tt.wantError {
			if err == nil {
				t.Errorf("Seek(%d, %d) should fail", tt.offset, tt.whence)
			}
			continue
		}
		if err != nil {
			t.Errorf("Seek(%d, %d) failed: %v", tt.offset, tt.whence, err)
			continue
		}
		if pos != tt.wantPos {
			t.Errorf("Seek(%d, %d) position: got %d, want %d",
				tt.offset, tt.whence, pos, tt.wantPos)
		}
	}
}

func TestEndOfFile(t *testing.T) {
	disk := NewDiskImage()
	f, err := disk.OpenFile("TEST.DAT", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write small amount of data
	data := []byte("test data")
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	f.Close()

	// Try to read past EOF
	f, err = disk.OpenFile("TEST.DAT", false)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	buf := make([]byte, 100)
	n, err := f.Read(buf)
	if err != io.EOF {
		t.Errorf("Expected EOF error, got %v", err)
	}
	if n != len(data) {
		t.Errorf("Wrong read size at EOF: got %d, want %d", n, len(data))
	}

	// Try to seek past EOF
	_, err = f.Seek(100, io.SeekStart)
	if err != nil {
		t.Errorf("Seek past EOF should succeed: %v", err)
	}

	n, err = f.Read(buf)
	if err != io.EOF {
		t.Errorf("Read past EOF should return EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("Read past EOF returned data: %d bytes", n)
	}
}

func TestReadOnly(t *testing.T) {
	disk := NewDiskImage()
	
	// Create and write to file
	f, err := disk.OpenFile("TEST.DAT", true)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	
	data := []byte("test data")
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	f.Close()

	// Set read-only attribute
	entry, _, err := disk.directory.FindFile("TEST.DAT")
	if err != nil {
		t.Fatalf("Failed to find file: %v", err)
	}

	attrs := &FileAttributes{ReadOnly: true}
	attrs.ApplyToDirectoryEntry(entry)

	// Try to write to read-only file
	f, err = disk.OpenFile("TEST.DAT", false)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	_, err = f.Write([]byte("new data"))
	if err == nil {
		t.Error("Write to read-only file should fail")
	}

	_, err = f.WriteAt([]byte("new data"), 0)
	if err == nil {
		t.Error("WriteAt to read-only file should fail")
	}
}