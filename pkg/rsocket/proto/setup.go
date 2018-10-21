package proto

const defaultMetadataMimeType = "application/json"
const defaultDataMimeType = "application/binary"

// SetupOptions configures the SETUP frame
type SetupOption struct {
	Version          Version  // Version number of the protocol.
	Lease            bool     // Will honor LEASE (or not).
	ResumeToken      Token    // Token used for client resume identification.
	MetadataMimeType string   // MIME Type for encoding of Metadata.
	DataMimeType     string   // MIME Type for encoding of Data.
	Payload          *Payload // Payload describing connection capabilities of the endpoint sending the Setup header.
}

func NewSetupOption() *SetupOption {
	return &SetupOption{
		LatestVersion,
		false,
		nil,
		defaultMetadataMimeType,
		defaultDataMimeType,
		new(Payload),
	}
}
