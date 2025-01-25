// file: pkg/diskimg/fileattr.go

package diskimg

// File attribute bit positions from +3DOS spec
const (
	// Type field (t1-t3) attributes
	AttrReadOnly = 0x01 // t1: 0=read-write, 1=read-only
	AttrSystem   = 0x02 // t2: 0=not system, 1=system file
	AttrArchived = 0x20 // t3: 0=not archived, 1=archived

	// Name field (f1-f8) attributes - f5-f8 reserved, always 0
	AttrUserF1   = 0x40 // f1: User-defined
	AttrUserF2   = 0x80 // f2: User-defined
	AttrUserF3   = 0x40 // f3: User-defined
	AttrUserF4   = 0x80 // f4: User-defined
)

// FileAttributes represents +3DOS file attributes
type FileAttributes struct {
	// Type field attributes
	ReadOnly bool // t1
	System   bool // t2
	Archived bool // t3

	// User-defined attributes
	UserF1   bool // f1
	UserF2   bool // f2
	UserF3   bool // f3
	UserF4   bool // f4
}

// GetTypeAttributes returns the type field attributes as a byte
func (fa *FileAttributes) GetTypeAttributes() byte {
	var attr byte
	if fa.ReadOnly {
		attr |= AttrReadOnly
	}
	if fa.System {
		attr |= AttrSystem
	}
	if fa.Archived {
		attr |= AttrArchived
	}
	return attr
}

// SetTypeAttributes sets attributes from type field byte
func (fa *FileAttributes) SetTypeAttributes(b byte) {
	fa.ReadOnly = (b & AttrReadOnly) != 0
	fa.System = (b & AttrSystem) != 0
	fa.Archived = (b & AttrArchived) != 0
}

// GetNameAttributes returns the name field attributes as a byte array
func (fa *FileAttributes) GetNameAttributes() [8]byte {
	var attrs [8]byte
	// Only f1-f4 are used, f5-f8 are reserved (always 0)
	if fa.UserF1 {
		attrs[0] |= AttrUserF1
	}
	if fa.UserF2 {
		attrs[1] |= AttrUserF2
	}
	if fa.UserF3 {
		attrs[2] |= AttrUserF3
	}
	if fa.UserF4 {
		attrs[3] |= AttrUserF4
	}
	return attrs
}

// SetNameAttributes sets attributes from name field bytes
func (fa *FileAttributes) SetNameAttributes(attrs [8]byte) {
	// Only process f1-f4, ignore f5-f8 (reserved)
	fa.UserF1 = (attrs[0] & AttrUserF1) != 0
	fa.UserF2 = (attrs[1] & AttrUserF2) != 0
	fa.UserF3 = (attrs[2] & AttrUserF3) != 0
	fa.UserF4 = (attrs[3] & AttrUserF4) != 0
}

// ApplyToDirectoryEntry applies attributes to a directory entry
func (fa *FileAttributes) ApplyToDirectoryEntry(entry *DirectoryEntry) {
	// Apply type attributes to extension bytes
	typeAttrs := fa.GetTypeAttributes()
	for i := range entry.Extension {
		if (typeAttrs & (1 << uint(i))) != 0 {
			entry.Extension[i] |= 0x80 // Set high bit
		} else {
			entry.Extension[i] &= 0x7F // Clear high bit
		}
	}

	// Apply name attributes
	nameAttrs := fa.GetNameAttributes()
	for i := range entry.Name {
		if (nameAttrs[i] & 0x80) != 0 {
			entry.Name[i] |= 0x80 // Set high bit
		} else {
			entry.Name[i] &= 0x7F // Clear high bit
		}
	}
}

// ReadFromDirectoryEntry extracts attributes from a directory entry
func (fa *FileAttributes) ReadFromDirectoryEntry(entry *DirectoryEntry) {
	// Extract type attributes from extension bytes
	var typeAttrs byte
	for i := range entry.Extension {
		if (entry.Extension[i] & 0x80) != 0 {
			typeAttrs |= 1 << uint(i)
		}
	}
	fa.SetTypeAttributes(typeAttrs)

	// Extract name attributes
	var nameAttrs [8]byte
	for i := range entry.Name {
		if (entry.Name[i] & 0x80) != 0 {
			nameAttrs[i] = 0x80
		}
	}
	fa.SetNameAttributes(nameAttrs)
}