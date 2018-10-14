package frame

type Flags uint16

const (
	FlagIgnore           Flags = 0x0200 // Ignore frame if not understood
	FlagMetadata         Flags = 0x0100 // Metadata present
	FlagResumeEnable     Flags = 0x0080 // Client requests resume capability if possible. Resume Identification Token present.
	FlagLease            Flags = 0x0040 // Will honor LEASE (or not).
	FlagKeepaliveRespond Flags = 0x0080 // Respond with KEEPALIVE or not
	FlagFollows          Flags = 0x0080 // More fragments follow this fragment.
	FlagComplete         Flags = 0x0040 // Indicate stream completion.
	FlagNext             Flags = 0x0020 // Indicate Next (Payload Data and/or Metadata present).
)

func (flags Flags) Set(flag Flags) {
	flags |= flag
}

func (flags Flags) IsSet(flag Flags) bool {
	return flags&flag == flag
}
