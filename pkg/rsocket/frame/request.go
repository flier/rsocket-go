package frame

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

const initReqsSize = uint32Size
const reqsSize = uint32Size

// RequestResponseFrame sent by client to request a single response.
type RequestResponseFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

// NewRequestResponseFrame create a new RequestResponseFrame.
func NewRequestResponseFrame(
	streamID StreamID,
	follows bool,
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
		&Header{streamID, TypeRequestResponse, flags},
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

// Size returns the encoded size of the frame.
func (request *RequestResponseFrame) Size() int {
	return request.Header.Size() + request.Metadata.Size() + len(request.Data)
}

// WriteTo writes the encoded frame to w.
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

// RequestFireAndForgetFrame sent by client to request a single one-way message.
type RequestFireAndForgetFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

// NewRequestFireAndForgetFrame creates a new RequestFireAndForgetFrame.
func NewRequestFireAndForgetFrame(
	streamID StreamID,
	follows bool,
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
		&Header{streamID, TypeRequestFireAndForget, flags},
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

// Size returns the encoded size of the frame.
func (request *RequestFireAndForgetFrame) Size() int {
	return request.Header.Size() + request.Metadata.Size() + len(request.Data)
}

// WriteTo writes the encoded frame to w.
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

// RequestStreamFrame sent by client to request a completable stream.
type RequestStreamFrame struct {
	*Header
	InitialRequests uint32
	Metadata        Metadata
	Data            []byte
}

// NewRequestStreamFrame creates a new RequestStreamFrame.
func NewRequestStreamFrame(
	streamID StreamID,
	follows bool,
	initReqs uint32,
	hasMetadata bool,
	metadata Metadata,
	data []byte,
) *RequestStreamFrame {
	var flags Flags

	if hasMetadata {
		flags.Set(FlagMetadata)
	}
	if follows {
		flags.Set(FlagFollows)
	}

	return &RequestStreamFrame{
		&Header{streamID, TypeRequestStream, flags},
		initReqs,
		metadata,
		data,
	}
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

// Size returns the encoded size of the frame.
func (request *RequestStreamFrame) Size() int {
	return request.Header.Size() + initReqsSize + request.Metadata.Size() + len(request.Data)
}

// WriteTo writes the encoded frame to w.
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

// RequestChannelFrame sent by client to request a completable stream in both directions.
type RequestChannelFrame struct {
	*Header
	InitialRequests uint32
	Metadata        Metadata
	Data            []byte
}

// NewRequestChannelFrame creates a new RequestChannelFrame.
func NewRequestChannelFrame(
	streamID StreamID,
	follows bool,
	complete bool,
	initReqs uint32,
	hasMetadata bool,
	metadata Metadata,
	data []byte,
) *RequestChannelFrame {
	var flags Flags

	if hasMetadata {
		flags.Set(FlagMetadata)
	}
	if follows {
		flags.Set(FlagFollows)
	}
	if complete {
		flags.Set(FlagComplete)
	}

	return &RequestChannelFrame{
		&Header{streamID, TypeRequestChannel, flags},
		initReqs,
		metadata,
		data,
	}
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

func (request *RequestChannelFrame) String() string {
	if request.Complete() {
		return fmt.Sprintf("%s[COMPLETE]", request.Type())
	}

	return request.Type().String()
}

// Size returns the encoded size of the frame.
func (request *RequestChannelFrame) Size() int {
	return request.Header.Size() + initReqsSize + request.Metadata.Size() + len(request.Data)
}

// WriteTo writes the encoded frame to w.
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

// RequestNFrame sent to request N more items with Reactive Streams semantics.
type RequestNFrame struct {
	*Header
	N uint32
}

// NewRequestNFrame creates a new RequestNFrame.
func NewRequestNFrame(streamID StreamID, requests uint32) *RequestNFrame {
	return &RequestNFrame{
		&Header{streamID, TypeRequestN, 0},
		requests,
	}
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

func (request *RequestNFrame) String() string {
	return fmt.Sprintf("REQUEST_N[%d]", request.N)
}

// Size returns the encoded size of the frame.
func (request *RequestNFrame) Size() int {
	return request.Header.Size() + reqsSize
}

// WriteTo writes the encoded frame to w.
func (request *RequestNFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = request.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, request.N); err != nil {
		return
	}

	wrote += reqsSize

	return
}
