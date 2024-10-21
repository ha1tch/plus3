package main

import (
	"fmt"
	"github.com/ha1tch/plus3/pkg/disk"
	"log"
)

func main() {
	// Create a new +3DOS disk image with default settings
	newDisk, err := disk.NewDisk()
	if err != nil {
		log.Fatalf("Failed to create new disk: %v", err)
	}

	// Print out the disk details to verify the creation process
	fmt.Println("New +3DOS Disk Image Created")
	newDisk.PrintDetails()
}

