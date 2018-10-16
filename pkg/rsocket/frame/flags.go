package frame

// Flags of frame
type Flags uint16

const (
	// FlagIgnore indicates the protocol can ignore frame if not understood
	FlagIgnore Flags = 0x0200
	// FlagMetadata indicates the metadata present
	FlagMetadata Flags = 0x0100
	// FlagResumeEnable indicates the client requests resume capability if possible. Resume Identification Token present.
	FlagResumeEnable Flags = 0x0080
	// FlagLease indicates the requester will honor LEASE (or not).
	FlagLease Flags = 0x0040
	// FlagKeepaliveRespond indicates respond with KEEPALIVE or not.
	FlagKeepaliveRespond Flags = 0x0080
	// FlagFollows indicates more fragments follow this fragment.
	FlagFollows Flags = 0x0080
	// FlagComplete indicates stream completion.
	FlagComplete Flags = 0x0040
	// FlagNext indicates Next (Payload Data and/or Metadata present).
	FlagNext Flags = 0x0020
)

// Set flag
func (flags *Flags) Set(flag Flags) {
	*flags |= flag
}

// IsSet returns the flag is set or not.
func (flags *Flags) IsSet(flag Flags) bool {
	return *flags&flag == flag
}
