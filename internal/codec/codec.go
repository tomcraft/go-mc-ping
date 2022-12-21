package codec

import (
	"_tomcraft/go-mc-ping/internal/types"
	"encoding/binary"
	"encoding/json"
	"errors"
	"golang.org/x/exp/constraints"
	"io"
	"math"
	"reflect"
	"sync"
)

type ByteArrayReader interface {
	io.Reader
	io.ByteReader
}

type ByteArrayWriter interface {
	io.Writer
	io.ByteWriter
}

type TypeCodec struct {
	Writer func(writer ByteArrayWriter, value reflect.Value) error
	Reader func(reader ByteArrayReader, value reflect.Value) error
}

type PacketCodec []*TypeCodec

var typeCodecs = make(map[string]*TypeCodec)
var packetCodecs = make(map[reflect.Type]*PacketCodec)
var registrationLock sync.Mutex

func RegisterCodec(name string, codec TypeCodec) {
	typeCodecs[name] = &codec
}

func RegisterPacketCodec(packetType reflect.Type) (*PacketCodec, error) {
	numFields := packetType.NumField()
	codecs := make(PacketCodec, numFields)
	for i := 0; i < numFields; i++ {
		field := packetType.Field(i)
		tag := field.Tag.Get("packet")
		if tag == "" || tag == "-" {
			continue
		}
		codec, ok := typeCodecs[tag]
		if !ok {
			return &codecs, errors.New("unknown serializer type: " + tag)
		}
		codecs[i] = codec
	}
	packetCodecs[packetType] = &codecs
	return &codecs, nil
}

func GetOrRegisterPacketCodec(packetType reflect.Type) (*PacketCodec, error) {
	codec, ok := packetCodecs[packetType]
	if !ok {
		registrationLock.Lock()
		defer registrationLock.Unlock()
		return RegisterPacketCodec(packetType)
	}
	return codec, nil
}

func ReadInt8(reader ByteArrayReader) (int8, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	return int8(b), nil
}

func WriteInt8(writer ByteArrayWriter, value int8) error {
	return writer.WriteByte(byte(value))
}

func ReadByte(reader ByteArrayReader) (byte, error) {
	return reader.ReadByte()
}

func WriteByte(writer ByteArrayWriter, value byte) error {
	return writer.WriteByte(value)
}

func ReadVarInt(reader ByteArrayReader) (int, error) {
	value, err := binary.ReadVarint(reader)
	return int(value), err
}

func ReadUnsignedVarInt(reader ByteArrayReader) (uint, error) {
	value, err := binary.ReadUvarint(reader)
	return uint(value), err
}

func _writeVarInt(writer ByteArrayWriter, x int64) error {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return _writeUnsignedVarInt(writer, ux)
}

func _writeUnsignedVarInt(writer ByteArrayWriter, x uint64) error {
	for x >= 0x80 {
		if err := writer.WriteByte(byte(x) | 0x80); err != nil {
			return err
		}
		x >>= 7
	}
	return writer.WriteByte(byte(x))
}

func WriteUnsignedVarInt(writer ByteArrayWriter, value uint) error {
	return _writeUnsignedVarInt(writer, uint64(value))
}

func WriteVarInt(writer ByteArrayWriter, value int) error {
	return _writeVarInt(writer, int64(value))
}

func readAnyInt[T constraints.Signed](reader ByteArrayReader, size int) (T, error) {
	ux, err := readAnyUnsignedInt[uint64](reader, size)
	if err != nil {
		return 0, err
	}
	x := int64(ux >> 1)
	if ux&1 != 0 {
		x = ^x
	}
	return T(x), err
}

func readAnyUnsignedInt[T constraints.Unsigned](reader ByteArrayReader, size int) (T, error) {
	var val uint64
	for i := 0; i < size; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, nil
		}
		if i != 0 {
			val <<= 8
		}
		val |= uint64(b)
	}
	return T(val), nil
}

func writeAnyInt[T constraints.Signed](writer ByteArrayWriter, value T, size int) error {
	ux := uint64(value) << 1
	if value < 0 {
		ux = ^ux
	}
	return writeAnyUnsignedInt(writer, ux, size)
}

func writeAnyUnsignedInt[T constraints.Unsigned](writer ByteArrayWriter, value T, size int) error {
	for i := size - 1; i >= 0; i-- {
		if err := writer.WriteByte(byte(value >> (i * 8))); err != nil {
			return err
		}
	}
	return nil
}

func ReadInt(reader ByteArrayReader) (int, error) {
	return readAnyInt[int](reader, 4)
}

func WriteInt(writer ByteArrayWriter, value int) error {
	return writeAnyInt[int](writer, value, 4)
}

func ReadInt16(reader ByteArrayReader) (int16, error) {
	return readAnyInt[int16](reader, 2)
}

func WriteInt16(writer ByteArrayWriter, value int16) error {
	return writeAnyInt[int16](writer, value, 2)
}

func ReadInt64(reader ByteArrayReader) (int64, error) {
	return readAnyInt[int64](reader, 8)
}

func WriteInt64(writer ByteArrayWriter, value int64) error {
	return writeAnyInt[int64](writer, value, 8)
}

func ReadFloat(reader ByteArrayReader) (float32, error) {
	val, err := readAnyUnsignedInt[uint32](reader, 4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(val), nil
}

func WriteFloat(writer ByteArrayWriter, value float32) error {
	return writeAnyUnsignedInt[uint32](writer, math.Float32bits(value), 4)
}

func ReadFloat64(reader ByteArrayReader) (float64, error) {
	val, err := readAnyUnsignedInt[uint64](reader, 8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(val), nil
}

func WriteFloat64(writer ByteArrayWriter, value float64) error {
	return writeAnyUnsignedInt[uint64](writer, math.Float64bits(value), 8)
}

func ReadString(reader ByteArrayReader) (string, error) {
	if val, err := ReadByteArray(reader); err != nil {
		return "", err
	} else {
		return string(val), nil
	}
}

func WriteString(writer ByteArrayWriter, value string) error {
	return WriteByteArray(writer, []byte(value))
}

func ReadByteArray(reader ByteArrayReader) ([]byte, error) {
	length, err := ReadUnsignedVarInt(reader)
	if err != nil {
		return nil, err
	}
	byteArray := make([]byte, length)
	_, err = reader.Read(byteArray)
	return byteArray, err
}

func WriteByteArray(writer ByteArrayWriter, value []byte) error {
	if err := WriteUnsignedVarInt(writer, uint(len(value))); err != nil {
		return err
	}
	_, err := writer.Write(value)
	return err
}

func ReadBool(reader ByteArrayReader) (bool, error) {
	val, err := reader.ReadByte()
	if err != nil {
		return false, err
	}
	if val == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func WriteBool(writer ByteArrayWriter, value bool) error {
	if value {
		return writer.WriteByte(1)
	} else {
		return writer.WriteByte(0)
	}
}

func ReadChatComponent(reader ByteArrayReader) (*types.ChatComponent, error) {
	return ReadChatComponentTo(reader, new(types.ChatComponent))
}

func ReadChatComponentTo(reader ByteArrayReader, componentPtr *types.ChatComponent) (*types.ChatComponent, error) {
	return ReadJson[types.ChatComponent](reader, componentPtr)
}

func WriteChatComponent(writer ByteArrayWriter, value types.ChatComponent) error {
	return WriteJson(writer, value)
}

func ReadJson[T any](reader ByteArrayReader, value *T) (*T, error) {
	array, err := ReadByteArray(reader)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(array, value); err != nil {
		return nil, err
	}
	return value, nil
}

func WriteJson(writer ByteArrayWriter, value any) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return WriteByteArray(writer, jsonBytes)
}

func SerializePacket(writer ByteArrayWriter, packet any) error {
	packetValue := reflect.ValueOf(packet)
	if packetValue.Kind() == reflect.Ptr {
		packetValue = packetValue.Elem()
	}
	packetType := packetValue.Type()
	packetCodec, err := GetOrRegisterPacketCodec(packetType)
	if err != nil {
		return err
	}
	for i, codec := range *packetCodec {
		if codec == nil {
			continue
		}
		fieldValue := packetValue.Field(i)
		if err := codec.Writer(writer, fieldValue); err != nil {
			return err
		}
	}
	return nil
}

func DeserializePacket[T any](reader ByteArrayReader, packetType reflect.Type) (*T, error) {
	packetPtr := new(T)
	packetCodec, err := GetOrRegisterPacketCodec(packetType)
	if err != nil {
		return nil, err
	}
	packetValue := reflect.ValueOf(packetPtr).Elem()
	for i, codec := range *packetCodec {
		if codec == nil {
			continue
		}
		fieldValue := packetValue.Field(i)
		if err := codec.Reader(reader, fieldValue); err != nil {
			return nil, err
		}
	}
	return packetPtr, nil
}

func createWriter[T any](writeFn func(ByteArrayWriter, T) error, getterFn func(reflect.Value) T) func(ByteArrayWriter, reflect.Value) error {
	return func(writer ByteArrayWriter, value reflect.Value) error {
		return writeFn(writer, getterFn(value))
	}
}

func createReader[T any](readFn func(ByteArrayReader) (T, error), setterFn func(reflect.Value, T)) func(ByteArrayReader, reflect.Value) error {
	return func(reader ByteArrayReader, value reflect.Value) error {
		val, err := readFn(reader)
		if err != nil {
			return err
		}
		setterFn(value, val)
		return nil
	}
}

func createCodec[T any](writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error), getterFn func(reflect.Value) T, setterFn func(reflect.Value, T)) TypeCodec {
	return TypeCodec{
		Writer: createWriter[T](writeFn, getterFn),
		Reader: createReader[T](readFn, setterFn),
	}
}

func createIntCodec[T constraints.Signed](writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error)) TypeCodec {
	return TypeCodec{
		Writer: func(writer ByteArrayWriter, value reflect.Value) error {
			return writeFn(writer, T(value.Int()))
		},
		Reader: createReader[T](readFn, func(value reflect.Value, val T) {
			value.SetInt(int64(val))
		}),
	}
}

func createUnsignedIntCodec[T constraints.Unsigned](writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error)) TypeCodec {
	return TypeCodec{
		Writer: func(writer ByteArrayWriter, value reflect.Value) error {
			return writeFn(writer, T(value.Uint()))
		},
		Reader: createReader[T](readFn, func(value reflect.Value, val T) {
			value.SetUint(uint64(val))
		}),
	}
}

func createFloatCodec[T constraints.Float](writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error)) TypeCodec {
	return TypeCodec{
		Writer: func(writer ByteArrayWriter, value reflect.Value) error {
			return writeFn(writer, T(value.Float()))
		},
		Reader: createReader[T](readFn, func(value reflect.Value, val T) {
			value.SetFloat(float64(val))
		}),
	}
}

func init() {
	RegisterCodec("byte", createUnsignedIntCodec[byte](WriteByte, ReadByte))
	RegisterCodec("int8", createIntCodec[int8](WriteInt8, ReadInt8))
	RegisterCodec("int16", createIntCodec[int16](WriteInt16, ReadInt16))
	RegisterCodec("int", createIntCodec[int](WriteInt, ReadInt))
	RegisterCodec("int64", createCodec[int64](WriteInt64, ReadInt64, reflect.Value.Int, reflect.Value.SetInt))
	RegisterCodec("varint", createIntCodec[int](WriteVarInt, ReadVarInt))
	RegisterCodec("uvarint", createUnsignedIntCodec[uint](WriteUnsignedVarInt, ReadUnsignedVarInt))

	RegisterCodec("float", createFloatCodec[float32](WriteFloat, ReadFloat))
	RegisterCodec("float64", createCodec[float64](WriteFloat64, ReadFloat64, reflect.Value.Float, reflect.Value.SetFloat))

	RegisterCodec("bool", createCodec[bool](WriteBool, ReadBool, reflect.Value.Bool, reflect.Value.SetBool))
	RegisterCodec("string", createCodec[string](WriteString, ReadString, reflect.Value.String, reflect.Value.SetString))
	RegisterCodec("byte_array", createCodec[[]byte](WriteByteArray, ReadByteArray, reflect.Value.Bytes, reflect.Value.SetBytes))

	RegisterCodec("component", TypeCodec{
		Writer: createWriter(WriteChatComponent, func(value reflect.Value) types.ChatComponent {
			return value.Interface().(types.ChatComponent)
		}),
		Reader: func(reader ByteArrayReader, value reflect.Value) error {
			val := value.Addr().Interface().(*types.ChatComponent)
			_, err := ReadChatComponentTo(reader, val)
			return err
		},
	})
	RegisterCodec("json", TypeCodec{
		Writer: createWriter(WriteJson, reflect.Value.Interface),
		Reader: func(reader ByteArrayReader, value reflect.Value) error {
			val := value.Addr().Interface()
			_, err := ReadJson(reader, &val)
			return err
		},
	})
}
