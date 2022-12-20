package internal

import (
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

func readInt8(reader ByteArrayReader) (int8, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	return int8(b), nil
}

func writeInt8(writer ByteArrayWriter, value int8) error {
	return writer.WriteByte(byte(value))
}

func readByte(reader ByteArrayReader) (byte, error) {
	return reader.ReadByte()
}

func writeByte(writer ByteArrayWriter, value byte) error {
	return writer.WriteByte(value)
}

func readVarInt(reader ByteArrayReader) (int, error) {
	value, err := binary.ReadUvarint(reader)
	return int(value), err
}

func writeVarInt(writer ByteArrayWriter, value int) error {
	for value >= 0x80 {
		if err := writer.WriteByte(byte(value) | 0x80); err != nil {
			return err
		}
		value >>= 7
	}
	return writer.WriteByte(byte(value))
}

func readAnyInt[T constraints.Integer](reader ByteArrayReader, size int) (T, error) {
	var val uint64
	for i := 0; i < size; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, nil
		}
		val |= uint64(b&0xFF) << i * 8
	}
	return T(val), nil
}

func writeAnyInt[T constraints.Integer](writer ByteArrayWriter, value T, size int) error {
	for i := 0; i < size; i++ {
		if err := writer.WriteByte(byte(value)); err != nil {
			return err
		}
		value >>= 8
	}
	return nil
}

func readInt(reader ByteArrayReader) (int, error) {
	return readAnyInt[int](reader, 4)
}

func writeInt(writer ByteArrayWriter, value int) error {
	return writeAnyInt[int](writer, value, 4)
}

func readInt16(reader ByteArrayReader) (int16, error) {
	return readAnyInt[int16](reader, 2)
}

func writeInt16(writer ByteArrayWriter, value int16) error {
	return writeAnyInt[int16](writer, value, 2)
}

func readInt64(reader ByteArrayReader) (int64, error) {
	return readAnyInt[int64](reader, 8)
}

func writeInt64(writer ByteArrayWriter, value int64) error {
	return writeAnyInt[int64](writer, value, 8)
}

func readFloat(reader ByteArrayReader) (float32, error) {
	val, err := readAnyInt[uint32](reader, 4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(val), nil
}

func writeFloat(writer ByteArrayWriter, value float32) error {
	return writeAnyInt[uint32](writer, math.Float32bits(value), 4)
}

func readFloat64(reader ByteArrayReader) (float64, error) {
	val, err := readAnyInt[uint64](reader, 8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(val), nil
}

func writeFloat64(writer ByteArrayWriter, value float64) error {
	return writeAnyInt[uint64](writer, math.Float64bits(value), 8)
}

func readString(reader ByteArrayReader) (string, error) {
	if val, err := readByteArray(reader); err != nil {
		return "", err
	} else {
		return string(val), nil
	}
}

func writeString(writer ByteArrayWriter, value string) error {
	return writeByteArray(writer, []byte(value))
}

func readByteArray(reader ByteArrayReader) ([]byte, error) {
	length, err := readVarInt(reader)
	if err != nil {
		return nil, err
	}
	byteArray := make([]byte, length)
	_, err = reader.Read(byteArray)
	return byteArray, err
}

func writeByteArray(writer ByteArrayWriter, value []byte) error {
	if err := writeVarInt(writer, len(value)); err != nil {
		return err
	}
	_, err := writer.Write(value)
	return err
}

func readBool(reader ByteArrayReader) (bool, error) {
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

func writeBool(writer ByteArrayWriter, value bool) error {
	if value {
		return writer.WriteByte(1)
	} else {
		return writer.WriteByte(0)
	}
}

func readChatComponent(reader ByteArrayReader) (ChatComponent, error) {
	return readJson(reader, ChatComponent{})
}

func writeChatComponent(writer ByteArrayWriter, value ChatComponent) error {
	return writeJson(writer, value)
}

func readJson[T any](reader ByteArrayReader, value T) (T, error) {
	array, err := readByteArray(reader)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(array, value); err != nil {
		return value, err
	}
	return value, nil
}

func writeJson(writer ByteArrayWriter, value any) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeByteArray(writer, jsonBytes)
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

func DeserializePacket(reader ByteArrayReader, packetType reflect.Type) (reflect.Value, error) {
	packetPtr := reflect.New(packetType)
	packetCodec, err := GetOrRegisterPacketCodec(packetType)
	if err != nil {
		return packetPtr, err
	}
	packetValue := packetPtr.Elem()
	for i, codec := range *packetCodec {
		if codec == nil {
			continue
		}
		fieldValue := packetValue.Field(i)
		if err := codec.Reader(reader, fieldValue); err != nil {
			return packetPtr, err
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
	RegisterCodec("byte", createUnsignedIntCodec[byte](writeByte, readByte))
	RegisterCodec("int8", createIntCodec[int8](writeInt8, readInt8))
	RegisterCodec("int16", createIntCodec[int16](writeInt16, readInt16))
	RegisterCodec("int", createIntCodec[int](writeInt, readInt))
	RegisterCodec("int64", createCodec[int64](writeInt64, readInt64, reflect.Value.Int, reflect.Value.SetInt))
	RegisterCodec("varint", createIntCodec[int](writeVarInt, readVarInt))

	RegisterCodec("float", createFloatCodec[float32](writeFloat, readFloat))
	RegisterCodec("float64", createCodec[float64](writeFloat64, readFloat64, reflect.Value.Float, reflect.Value.SetFloat))

	RegisterCodec("bool", createCodec[bool](writeBool, readBool, reflect.Value.Bool, reflect.Value.SetBool))
	RegisterCodec("string", createCodec[string](writeString, readString, reflect.Value.String, reflect.Value.SetString))
	RegisterCodec("byte_array", createCodec[[]byte](writeByteArray, readByteArray, reflect.Value.Bytes, reflect.Value.SetBytes))

	RegisterCodec("component", TypeCodec{
		Writer: createWriter(writeChatComponent, func(value reflect.Value) ChatComponent {
			return value.Interface().(ChatComponent)
		}),
		Reader: createReader(readChatComponent, func(value reflect.Value, val ChatComponent) {
			value.Set(reflect.ValueOf(val))
		}),
	})
	RegisterCodec("json", TypeCodec{
		Writer: createWriter(writeJson, reflect.Value.Interface),
		Reader: func(reader ByteArrayReader, value reflect.Value) error {
			val, err := readJson(reader, value)
			if err != nil {
				return err
			}
			value.Set(val)
			return nil
		},
	})
}
