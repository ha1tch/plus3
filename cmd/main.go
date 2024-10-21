package main

import (
    "fmt"
    "github.com/ha1tch/plus3/pkg/disk"
    "log"
)

func main() {
    // Example: Creating a new +3DOS disk image
    d, err := disk.NewDisk()
    if err != nil {
        log.Fatalf("Failed to create new disk: %v", err)
    }

    fmt.Println("Created new +3DOS disk image")

    // Example: Writing data to a file in the disk image
    file, err := d.CreateFile("example.txt")
    if err != nil {
        log.Fatalf("Failed to create file: %v", err)
    }

    data := []byte("Hello, +3DOS World!")
    file.Write(data)
    file.Close()

    fmt.Println("Wrote file to disk")
}

