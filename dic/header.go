package dic

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Header version constants
const (
	// System dictionary versions
	SystemDictVersion1 uint64 = 0x7366d3f18bd111e7
	SystemDictVersion2 uint64 = 0xce9f011a92394434

	// User dictionary versions
	UserDictVersion1 uint64 = 0xa50f31188bd211e7
	UserDictVersion2 uint64 = 0x9fdeb5a90168d868
	UserDictVersion3 uint64 = 0xca9811756ff64fb0
)

// HeaderVersion represents the version of a dictionary header
type HeaderVersion interface {
	ToU64() uint64
	HasGrammar() bool
	HasSynonymGroupIds() bool
	IsSystemDict() bool
	IsUserDict() bool
	String() string
}

// SystemDictVersion represents system dictionary version
type SystemDictVersion int

const (
	SystemDictVersionV1 SystemDictVersion = iota
	SystemDictVersionV2
)

func (v SystemDictVersion) ToU64() uint64 {
	switch v {
	case SystemDictVersionV1:
		return SystemDictVersion1
	case SystemDictVersionV2:
		return SystemDictVersion2
	default:
		return 0
	}
}

func (v SystemDictVersion) HasGrammar() bool {
	return true // All system dictionary versions have grammar
}

func (v SystemDictVersion) HasSynonymGroupIds() bool {
	return v == SystemDictVersionV2
}

func (v SystemDictVersion) IsSystemDict() bool {
	return true
}

func (v SystemDictVersion) IsUserDict() bool {
	return false
}

func (v SystemDictVersion) String() string {
	switch v {
	case SystemDictVersionV1:
		return "SystemDictV1"
	case SystemDictVersionV2:
		return "SystemDictV2"
	default:
		return "Unknown SystemDict"
	}
}

// UserDictVersion represents user dictionary version
type UserDictVersion int

const (
	UserDictVersionV1 UserDictVersion = iota
	UserDictVersionV2
	UserDictVersionV3
)

func (v UserDictVersion) ToU64() uint64 {
	switch v {
	case UserDictVersionV1:
		return UserDictVersion1
	case UserDictVersionV2:
		return UserDictVersion2
	case UserDictVersionV3:
		return UserDictVersion3
	default:
		return 0
	}
}

func (v UserDictVersion) HasGrammar() bool {
	return v == UserDictVersionV2 || v == UserDictVersionV3
}

func (v UserDictVersion) HasSynonymGroupIds() bool {
	return v == UserDictVersionV3
}

func (v UserDictVersion) IsSystemDict() bool {
	return false
}

func (v UserDictVersion) IsUserDict() bool {
	return true
}

func (v UserDictVersion) String() string {
	switch v {
	case UserDictVersionV1:
		return "UserDictV1"
	case UserDictVersionV2:
		return "UserDictV2"
	case UserDictVersionV3:
		return "UserDictV3"
	default:
		return "Unknown UserDict"
	}
}

// HeaderVersionFromU64 creates a HeaderVersion from a uint64 value
func HeaderVersionFromU64(v uint64) (HeaderVersion, error) {
	switch v {
	case SystemDictVersion1:
		return SystemDictVersionV1, nil
	case SystemDictVersion2:
		return SystemDictVersionV2, nil
	case UserDictVersion1:
		return UserDictVersionV1, nil
	case UserDictVersion2:
		return UserDictVersionV2, nil
	case UserDictVersion3:
		return UserDictVersionV3, nil
	default:
		return nil, fmt.Errorf("invalid header version: unknown version: 0x%016x", v)
	}
}

// Header represents a dictionary header
type Header struct {
	Version     HeaderVersion
	CreateTime  uint64 // Unix timestamp
	Description string
}

const (
	// DescriptionSize is the fixed size of the description field in bytes
	DescriptionSize = 256
	// StorageSize is the total size of the header in bytes
	StorageSize = 8 + 8 + DescriptionSize // version + create_time + description
)

// NewHeader creates a new system dictionary header with current timestamp
func NewHeader() *Header {
	return &Header{
		Version:     SystemDictVersionV2,
		CreateTime:  uint64(time.Now().Unix()),
		Description: "",
	}
}

// NewHeaderWithVersion creates a new header with the specified version
func NewHeaderWithVersion(version HeaderVersion) *Header {
	return &Header{
		Version:     version,
		CreateTime:  uint64(time.Now().Unix()),
		Description: "",
	}
}

// SetTime sets the creation time and returns the previous time
func (h *Header) SetTime(t time.Time) time.Time {
	oldTime := time.Unix(int64(h.CreateTime), 0)
	h.CreateTime = uint64(t.Unix())
	return oldTime
}

// ParseHeader parses a header from binary data
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < StorageSize {
		return nil, fmt.Errorf("insufficient data to parse header: need %d bytes, have %d bytes", StorageSize, len(data))
	}

	reader := NewReader(data)

	// Read version
	versionRaw, err := reader.ReadU64()
	if err != nil {
		return nil, err
	}

	version, err := HeaderVersionFromU64(versionRaw)
	if err != nil {
		return nil, err
	}

	// Read create time
	createTime, err := reader.ReadU64()
	if err != nil {
		return nil, err
	}

	// Read description
	description, err := reader.ReadString(DescriptionSize)
	if err != nil {
		return nil, err
	}

	return &Header{
		Version:     version,
		CreateTime:  createTime,
		Description: description,
	}, nil
}

// WriteTo writes the header to binary format
func (h *Header) WriteTo(data []byte) error {
	if len(data) < StorageSize {
		return fmt.Errorf("insufficient data to write header: need %d bytes, have %d bytes", StorageSize, len(data))
	}

	if len(h.Description) > DescriptionSize {
		return fmt.Errorf("description too long: description length %d exceeds maximum %d", len(h.Description), DescriptionSize)
	}

	// Write version
	err := writeU64(data[0:8], h.Version.ToU64())
	if err != nil {
		return err
	}

	// Write create time
	err = writeU64(data[8:16], h.CreateTime)
	if err != nil {
		return err
	}

	// Write description
	copy(data[16:], []byte(h.Description))

	// Zero-fill remaining description bytes
	for i := len(h.Description); i < DescriptionSize; i++ {
		data[16+i] = 0
	}

	return nil
}

// Helper function to write uint64 in little-endian format
func writeU64(data []byte, value uint64) error {
	if len(data) < 8 {
		return fmt.Errorf("insufficient space to write uint64: required 8 bytes, got %d", len(data))
	}

	binary.LittleEndian.PutUint64(data, value)
	return nil
}

// String returns a string representation of the header
func (h *Header) String() string {
	createTime := time.Unix(int64(h.CreateTime), 0)
	return fmt.Sprintf("Header{Version: %s, CreateTime: %s, Description: %q}",
		h.Version.String(), createTime.Format("2006-01-02 15:04:05"), h.Description)
}
