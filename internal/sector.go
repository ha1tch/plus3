package internal

// TrackToSector converts a track and sector index into a linear sector index.
func TrackToSector(trackNum, sectorsPerTrack, side int) int {
    return trackNum*sectorsPerTrack + side
}

