package proto

import "google.golang.org/protobuf/encoding/protojson"

var marshaler = protojson.MarshalOptions{}

func (m *Test_Client_Call_DefaultError) Error() string {
	buf, _ := marshaler.Marshal(m)
	return string(buf)
}

func (m *Test_Client_Call_SpecialError) Error() string {
	buf, _ := marshaler.Marshal(m)
	return string(buf)
}
