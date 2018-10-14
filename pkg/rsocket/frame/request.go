package frame

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

const initReqsSize = uint32Size
const reqsSize = uint32Size

type RequestResponseFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

func NewRequestResponseFrame(streamId StreamId, follows bool,
	hasMetadata bool,
	metadata Metadata,
	data []byte,
) *RequestResponseFrame {
	var flags Flags

	if hasMetadata {
		flags.Set(FlagMetadata)
	}
	if follows {
		flags.Set(FlagFollows)
	}

	return &RequestResponseFrame{
		&Header{streamId, TypeRequestResponse, flags},
		metadata,
		data,
	}
}

func readRequestResponseFrame(r io.Reader, header *Header) (frame *RequestResponseFrame, err error) {
	var metadata, data []byte

	if header.HasMetadata() {
		if metadata, err = readMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &RequestResponseFrame{
		header,
		metadata,
		data,
	}

	return
}

func (request *RequestResponseFrame) Size() int {
	return request.Header.Size() + request.Metadata.Size() + len(request.Data)
}

func (request *RequestResponseFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = request.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if request.HasMetadata() {
		if n, err = request.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if n, err = writeExact(w, []byte(request.Data)); err != nil {
		return
	}

	wrote += n

	return
}

type RequestFireAndForgetFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

func NewRequestFireAndForgetFrame(streamId StreamId, follows bool,
	hasMetadata bool,
	metadata Metadata,
	data []byte,
) *RequestFireAndForgetFrame {
	var flags Flags

	if hasMetadata {
		flags.Set(FlagMetadata)
	}
	if follows {
		flags.Set(FlagFollows)
	}

	return &RequestFireAndForgetFrame{
		&Header{streamId, TypeRequestFireAndForget, flags},
		metadata,
		data,
	}
}

func readRequestFireAndForgetFrame(r io.Reader, header *Header) (frame *RequestFireAndForgetFrame, err error) {
	var metadata, data []byte

	if header.HasMetadata() {
		if metadata, err = readMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &RequestFireAndForgetFrame{
		header,
		metadata,
		data,
	}

	return
}

func (request *RequestFireAndForgetFrame) Size() int {
	return request.Header.Size() + request.Metadata.Size() + len(request.Data)
}

func (request *RequestFireAndForgetFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = request.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if request.HasMetadata() {
		if n, err = request.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if n, err = writeExact(w, []byte(request.Data)); err != nil {
		return
	}

	wrote += n

	return
}

type RequestStreamFrame struct {
	*Header
	InitialRequests uint32
	Metadata        Metadata
	Data            []byte
}

func readRequestStreamFrame(r io.Reader, header *Header) (frame *RequestStreamFrame, err error) {
	var initReqs uint32
	var metadata, data []byte

	if err = binary.Read(r, binary.BigEndian, &initReqs); err != nil {
		return
	}

	if header.HasMetadata() {
		if metadata, err = readMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &RequestStreamFrame{
		header,
		initReqs,
		metadata,
		data,
	}

	return
}

func (request *RequestStreamFrame) Size() int {
	return request.Header.Size() + initReqsSize + request.Metadata.Size() + len(request.Data)
}

func (request *RequestStreamFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = request.Header.WriteTo(w); err != nil {
		return
	}

	if err = binary.Write(w, binary.BigEndian, request.InitialRequests); err != nil {
		return
	}

	wrote += initReqsSize

	var n int64

	if request.HasMetadata() {
		if n, err = request.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if n, err = writeExact(w, []byte(request.Data)); err != nil {
		return
	}

	wrote += n

	return
}

type RequestChannelFrame struct {
	*Header
	InitialRequests uint32
	Metadata        Metadata
	Data            []byte
}

func readRequestChannelFrame(r io.Reader, header *Header) (frame *RequestChannelFrame, err error) {
	var initReqs uint32
	var metadata, data []byte

	if err = binary.Read(r, binary.BigEndian, &initReqs); err != nil {
		return
	}

	if header.HasMetadata() {
		if metadata, err = readMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &RequestChannelFrame{
		header,
		initReqs,
		metadata,
		data,
	}

	return
}

func (request *RequestChannelFrame) Size() int {
	return request.Header.Size() + initReqsSize + request.Metadata.Size() + len(request.Data)
}

func (request *RequestChannelFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = request.Header.WriteTo(w); err != nil {
		return
	}

	if err = binary.Write(w, binary.BigEndian, request.InitialRequests); err != nil {
		return
	}

	wrote += initReqsSize

	var n int64

	if request.HasMetadata() {
		if n, err = request.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	if n, err = writeExact(w, []byte(request.Data)); err != nil {
		return
	}

	wrote += n

	return
}

type RequestNFrame struct {
	*Header
	Requests uint32
}

func readRequestNFrame(r io.Reader, header *Header) (frame *RequestNFrame, err error) {
	var reqs uint32

	if err = binary.Read(r, binary.BigEndian, &reqs); err != nil {
		return
	}

	frame = &RequestNFrame{
		header,
		reqs,
	}

	return
}

func (request *RequestNFrame) Size() int {
	return request.Header.Size() + reqsSize
}

func (request *RequestNFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = request.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, request.Requests); err != nil {
		return
	}

	wrote += reqsSize

	return
}
