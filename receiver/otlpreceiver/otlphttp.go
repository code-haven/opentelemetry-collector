// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpreceiver

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"io"
	"io/ioutil"
)

// xProtobufMarshaler is a Marshaler which wraps runtime.ProtoMarshaller
// and sets ContentType to application/x-protobuf
type xProtobufMarshaler struct {
	*runtime.ProtoMarshaller
}

// ContentType always returns "application/x-protobuf".
func (*xProtobufMarshaler) ContentType() string {
	return "application/x-protobuf"
}

// NewDecoder returns a Decoder which reads proto stream from "reader".
func (marshaller *xProtobufMarshaler) NewDecoder(reader io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(value interface{}) error {
		var err error
		var gzipped bool

		reader, gzipped, err = isGzip(reader)
		if err != nil {
			return err
		}
		if gzipped {
			gzReader, err := gzip.NewReader(reader)
			if err != nil {
				return err
			}
			reader = gzReader
			defer gzReader.Close()
		}
		buffer, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		return marshaller.Unmarshal(buffer, value)
	})
}

// jSONMarshaller extends runtime.JSONPb to add support for gzipped payloads.
type jSONMarshaller struct {
	runtime.JSONPb
}

// NewDecoder returns a Decoder which reads JSON stream from "reader".
func (j *jSONMarshaller) NewDecoder(reader io.Reader) runtime.Decoder {
	var err error
	var gzipped bool

	reader, gzipped, err = isGzip(reader)
	errDecoder := func(decodeErr error) runtime.DecoderFunc {
		return func(value interface{}) error {
			return decodeErr
		}
	}
	if err != nil {
		return errDecoder(err)
	}
	if gzipped {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return errDecoder(err)
		}
		reader = gzReader
		defer gzReader.Close()
	}
	return runtime.DecoderWrapper{Decoder: json.NewDecoder(reader)}
}

// isGzip peaks into the first three bytes in the input stream and checks whether
// they match the standard gzip headers to confirm if it's gzipped.
func isGzip(input io.Reader) (io.Reader, bool, error) {
	const (
		gzipID1     = 0x1f
		gzipID2     = 0x8b
		gzipDeflate = 8
		peakLength  = 3
	)
	reader := bufio.NewReader(input)
	headerBytes, err := reader.Peek(peakLength)
	if err != nil {
		return reader, false, err
	}
	isGzip := headerBytes[0] == gzipID1 && headerBytes[1] == gzipID2 && headerBytes[2] == gzipDeflate
	return reader, isGzip, nil
}
