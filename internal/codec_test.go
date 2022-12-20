package internal

import (
	"bytes"
	"reflect"
	"testing"
)

func BenchmarkSerializePacket(b *testing.B) {
	packet := PlayerPosLookPacket{
		X: 140, Y: 200, Z: 90,
	}
	buffer := &bytes.Buffer{}
	buffer.Grow(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Reset()
		err := SerializePacket(buffer, packet)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeserializePacket(b *testing.B) {
	originalPacket := PlayerPosLookPacket{
		X: 140, Y: 200, Z: 90,
	}
	originalBuffer := &bytes.Buffer{}
	originalBuffer.Grow(1024)
	err := SerializePacket(originalBuffer, originalPacket)
	if err != nil {
		b.Fatal(err)
	}
	packetType := reflect.TypeOf(originalPacket)
	data := originalBuffer.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DeserializePacket(bytes.NewBuffer(data), packetType); err != nil {
			b.Fatal(err)
		}
	}
}
