package codec

import (
	"_tomcraft/go-mc-ping/internal/types"
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type TestFullPacket struct {
	A  byte                `packet:"byte"`
	B  int8                `packet:"int8"`
	C  int16               `packet:"int16"`
	D  int                 `packet:"int"`
	E  int64               `packet:"int64"`
	F  int                 `packet:"varint"`
	FF uint                `packet:"uvarint"`
	G  float32             `packet:"float"`
	H  float64             `packet:"float64"`
	I  bool                `packet:"bool"`
	J  string              `packet:"string"`
	K  types.ChatComponent `packet:"component"`
	L  types.Identity      `packet:"json"`
}

type TestPacket struct {
	X, Y, Z    float64 `packet:"float64"`
	Yaw, Pitch float32 `packet:"float"`
	Flags      byte    `packet:"byte"`
}

var buffer = new(bytes.Buffer)

func fn(i any) string {
	val := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	index := strings.LastIndex(val, "/") + 1
	return val[index:]
}

func bufferWrite[T any](t *testing.T, writeFn func(ByteArrayWriter, T) error, value T, expected []byte) {
	name := fmt.Sprintf("[%T] %s(%v)", value, fn(writeFn), value)
	t.Run(name, func(t *testing.T) {
		buffer.Reset()
		err := writeFn(buffer, value)
		if err != nil || !bytes.Equal(buffer.Bytes(), expected) {
			t.Fatalf(`%s = 0x%x, %v, want match for 0x%x, nil`, name, buffer.Bytes(), err, expected)
		}
	})
}

func testWriteRead[T comparable](t *testing.T, originalValue T, writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error)) {
	name := fmt.Sprintf("[%T] %s(%v) -> %s()", originalValue, fn(writeFn), originalValue, fn(readFn))
	t.Run(name, func(t *testing.T) {
		buffer.Reset()
		err := writeFn(buffer, originalValue)
		if err != nil {
			t.Fatal(name, " unexpected error at write: ", err)
		}
		val, err := readFn(buffer)
		if err != nil {
			t.Fatal(name, " unexpected error at read: ", err)
		}
		if originalValue != val {
			t.Fatalf("%s: expected %v but got %v", name, originalValue, val)
		}
	})
}

func testPacket[T comparable](t *testing.T, originalPacket T) {
	name := fmt.Sprintf("[%T] SerializePacket() -> DeserializePacket()", originalPacket)
	t.Run(name, func(t *testing.T) {
		buffer.Reset()
		err := SerializePacket(buffer, originalPacket)
		if err != nil {
			t.Fatal(name, " unexpected error at serialize: ", err)
		}
		packetPtr, err := DeserializePacket[T](buffer, reflect.TypeOf(originalPacket))
		if err != nil {
			t.Fatal(name, " unexpected error at deserialize: ", err)
		}
		decodedPacket := *packetPtr
		if originalPacket != decodedPacket {
			t.Fatalf("%s: expected %v but got %v", name, originalPacket, decodedPacket)
		}
	})
}

func TestWriteByte(t *testing.T) {
	bufferWrite[byte](t, WriteByte, 0, []byte{0x00})
	bufferWrite[byte](t, WriteByte, 56, []byte{0x38})
	bufferWrite[byte](t, WriteByte, 200, []byte{0xC8})
	bufferWrite[byte](t, WriteByte, 255, []byte{0xFF})
}

func TestWriteInt8(t *testing.T) {
	bufferWrite[int8](t, WriteInt8, 0, []byte{0x00})
	bufferWrite[int8](t, WriteInt8, 56, []byte{0x38})
	bufferWrite[int8](t, WriteInt8, -56, []byte{0xC8})
	bufferWrite[int8](t, WriteInt8, 127, []byte{0x7F})
	bufferWrite[int8](t, WriteInt8, -128, []byte{0x80})
}

func TestWriteReadInt(t *testing.T) {
	testWriteRead[byte](t, 248, WriteByte, ReadByte)
	testWriteRead[int8](t, 120, WriteInt8, ReadInt8)
	testWriteRead[int8](t, -120, WriteInt8, ReadInt8)
	testWriteRead[int16](t, 1024, WriteInt16, ReadInt16)
	testWriteRead[int16](t, -1024, WriteInt16, ReadInt16)
	testWriteRead[uint](t, 5890, WriteUnsignedVarInt, ReadUnsignedVarInt)
	testWriteRead[int](t, 4096, WriteVarInt, ReadVarInt)
	testWriteRead[int](t, -4096, WriteVarInt, ReadVarInt)
	testWriteRead[int](t, 8192, WriteInt, ReadInt)
	testWriteRead[int](t, -8192, WriteInt, ReadInt)
	testWriteRead[int64](t, 56000, WriteInt64, ReadInt64)
	testWriteRead[int64](t, -56000, WriteInt64, ReadInt64)
}

func TestWriteReadFloat(t *testing.T) {
	testWriteRead[float32](t, 3.14, WriteFloat, ReadFloat)
	testWriteRead[float32](t, -3.14, WriteFloat, ReadFloat)
	testWriteRead[float64](t, 30000.00000014, WriteFloat64, ReadFloat64)
	testWriteRead[float64](t, -30000.00000014, WriteFloat64, ReadFloat64)
}

func TestWriteReadOther(t *testing.T) {
	testWriteRead[string](t, "", WriteString, ReadString)
	testWriteRead[string](t, "testString", WriteString, ReadString)
	testWriteRead[bool](t, true, WriteBool, ReadBool)
	testWriteRead[bool](t, false, WriteBool, ReadBool)
}

func TestWriteReadJson(t *testing.T) {
	identity := types.Identity{Uuid: "417e3cae-eaaa-447f-8dae-2549e6aa60c6", Username: "_tomcraft"}
	buffer.Reset()
	err := WriteJson(buffer, identity)
	if err != nil {
		t.Fatal("unable to serialize struct: ", err)
	}
	newIdentityPtr, err := ReadJson(buffer, new(types.Identity))
	if err != nil {
		t.Fatal("unable to deserialize struct: ", err)
	}
	if *newIdentityPtr != identity {
		t.Fatalf("expected (%v), got (%v)", identity, *newIdentityPtr)
	}
}

func TestWriteReadChatComponent(t *testing.T) {
	component := types.ChatComponent{Text: "Test Message"}
	buffer.Reset()
	err := WriteChatComponent(buffer, component)
	if err != nil {
		t.Fatal("unable to serialize component: ", err)
	}
	newComponentPtr, err := ReadChatComponent(buffer)
	if err != nil {
		t.Fatal("unable to deserialize component: ", err)
	}
	if *newComponentPtr != component {
		t.Fatalf("expected (%v), got (%v)", component, *newComponentPtr)
	}
}

func TestSerializeDeserializePacket(t *testing.T) {
	packet1 := TestFullPacket{
		A:  0xBA,
		B:  125,
		C:  1000,
		D:  40000,
		E:  50000000,
		F:  -30000,
		FF: 30000,
		G:  900.5000,
		H:  90000.50000,
		I:  true,
		J:  "testStr",
		K:  types.ChatComponent{Text: "Text Message"},
		L:  types.Identity{Uuid: "417e3cae-eaaa-447f-8dae-2549e6aa60c6", Username: "_tomcraft"},
	}
	packet2 := TestPacket{
		X: 140, Y: 200, Z: 90,
	}
	testPacket(t, packet1)
	testPacket(t, packet2)
}

func BenchmarkSerializePacket(b *testing.B) {
	packet := TestPacket{
		X: 140, Y: 200, Z: 90,
	}
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
	originalPacket := TestPacket{
		X: 140, Y: 200, Z: 90,
	}
	originalBuffer := new(bytes.Buffer)
	originalBuffer.Grow(1024)
	err := SerializePacket(originalBuffer, originalPacket)
	if err != nil {
		b.Fatal(err)
	}
	packetType := reflect.TypeOf(originalPacket)
	data := originalBuffer.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DeserializePacket[TestPacket](bytes.NewBuffer(data), packetType); err != nil {
			b.Fatal(err)
		}
	}
}
