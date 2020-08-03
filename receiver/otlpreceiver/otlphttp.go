// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpreceiver

import (
	"compress/gzip"
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

func (m *xProtobufMarshaler) NewDecoder(reader io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(value interface{}) error {
		zReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		buffer, err := ioutil.ReadAll(zReader)
		if err != nil {
			return err
		}
		return m.Unmarshal(buffer, value)
	})
}