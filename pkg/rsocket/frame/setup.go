package frame

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"time"
)

const keepaliveSize = uint32Size
const maxLifetimeSize = uint32Size

type SetupFrame struct {
	*Header
	Version          Version
	Keepalive        time.Duration
	MaxLifetime      time.Duration
	ResumeToken      Token
	MetadataMimeType string
	DataMimeType     string
	Metadata         Metadata
	Data             []byte
}

func ReadSetupFrame(r io.Reader, header *Header) (frame *SetupFrame, err error) {
	var major, minor uint16
	var keepalive, maxLifetime uint32
	var resumeToken Token
	var metadataMimeType, dataMimeType string
	var metadata Metadata
	var data []byte

	if err = binary.Read(r, binary.BigEndian, &major); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &minor); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &keepalive); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &maxLifetime); err != nil {
		return
	}

	if header.HasResumeToken() {
		if resumeToken, err = ReadToken(r); err != nil {
			return
		}
	}

	var len byte

	if err = binary.Read(r, binary.BigEndian, &len); err != nil {
		return
	}

	buf := make([]byte, len)

	if err = binary.Read(r, binary.BigEndian, buf); err != nil {
		return
	}

	metadataMimeType = string(buf)

	if err = binary.Read(r, binary.BigEndian, &len); err != nil {
		return
	}

	buf = make([]byte, len)

	if err = binary.Read(r, binary.BigEndian, buf); err != nil {
		return
	}

	dataMimeType = string(buf)

	if header.HasMetadata() {
		if metadata, err = ReadMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &SetupFrame{
		header,
		Version{major, minor},
		time.Duration(keepalive) * time.Millisecond,
		time.Duration(maxLifetime) * time.Millisecond,
		resumeToken,
		metadataMimeType,
		dataMimeType,
		metadata,
		data,
	}

	return
}

func (setup *SetupFrame) Size() int {
	size := setup.Header.Size() + setup.Version.Size() + keepaliveSize + maxLifetimeSize

	if setup.HasResumeToken() {
		size += tokenLenSize + setup.ResumeToken.Size()
	}

	size += byteSize + len(setup.MetadataMimeType)
	size += byteSize + len(setup.DataMimeType)
	size += setup.Metadata.Size() + len(setup.Data)

	return size
}

func (setup *SetupFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = setup.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if n, err = setup.Version.WriteTo(w); err != nil {
		return
	}

	wrote += n

	if err = binary.Write(w, binary.BigEndian, uint32(setup.Keepalive/time.Millisecond)); err != nil {
		return
	}
	if err = binary.Write(w, binary.BigEndian, uint32(setup.MaxLifetime/time.Millisecond)); err != nil {
		return
	}

	wrote += keepaliveSize + maxLifetimeSize

	if setup.HasResumeToken() {
		if n, err = setup.ResumeToken.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if err = writeByte(w, byte(len(setup.MetadataMimeType))); err != nil {
		return
	}
	if n, err = writeExact(w, []byte(setup.MetadataMimeType)); err != nil {
		return
	}

	wrote += byteSize + n

	if err = writeByte(w, byte(len(setup.DataMimeType))); err != nil {
		return
	}
	if n, err = writeExact(w, []byte(setup.DataMimeType)); err != nil {
		return
	}

	wrote += byteSize + n

	if setup.HasMetadata() {
		if n, err = setup.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if n, err = writeExact(w, []byte(setup.Data)); err != nil {
		return
	}

	wrote += n

	return
}
