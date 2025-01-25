// file: cmd/create/create.go

package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

func Create(outPath string) error {
	disk := diskimg.NewDiskImage()
	if disk == nil {
		return fmt.Errorf("failed to create disk image")
	}

	// Ensure directory exists
	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	if err := disk.SaveToFile(outPath); err != nil {
		return fmt.Errorf("failed to save disk: %v", err)
	}

	return nil
}