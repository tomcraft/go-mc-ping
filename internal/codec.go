package internal

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"golang.org/x/exp/constraints"
	"io"
	"math"
	"reflect"
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

var typeCodecs = make(map[string]TypeCodec)

func RegisterCodec(name string, codec TypeCodec) {
	typeCodecs[name] = codec
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

func readInt(reader ByteArrayReader) (int, error) {
	array := make([]byte, 4)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(array)), nil
}

func writeInt(writer ByteArrayWriter, value int) error {
	array := make([]byte, 4)
	binary.BigEndian.PutUint32(array, uint32(value))
	_, err := writer.Write(array)
	return err
}

func readInt16(reader ByteArrayReader) (int16, error) {
	array := make([]byte, 2)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(array)), nil
}

func writeInt16(writer ByteArrayWriter, value int16) error {
	array := make([]byte, 2)
	binary.BigEndian.PutUint16(array, uint16(value))
	_, err := writer.Write(array)
	return err
}

func readInt64(reader ByteArrayReader) (int64, error) {
	array := make([]byte, 8)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(array)), nil
}

func writeInt64(writer ByteArrayWriter, value int64) error {
	array := make([]byte, 8)
	binary.BigEndian.PutUint64(array, uint64(value))
	_, err := writer.Write(array)
	return err
}

func readFloat(reader ByteArrayReader) (float32, error) {
	array := make([]byte, 4)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.BigEndian.Uint32(array)), nil
}

func writeFloat(writer ByteArrayWriter, value float32) error {
	array := make([]byte, 4)
	binary.BigEndian.PutUint32(array, math.Float32bits(value))
	_, err := writer.Write(array)
	return err
}

func readFloat64(reader ByteArrayReader) (float64, error) {
	array := make([]byte, 8)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(array)), nil
}

func writeFloat64(writer ByteArrayWriter, value float64) error {
	array := make([]byte, 8)
	binary.BigEndian.PutUint64(array, math.Float64bits(value))
	_, err := writer.Write(array)
	return err
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
	for i := 0; i < packetType.NumField(); i++ {
		// Get the field tag value
		field := packetType.Field(i)
		tag := field.Tag.Get("packet")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		fieldValue := packetValue.Field(i)
		codec, ok := typeCodecs[tag]
		if !ok {
			return errors.New("unknown serializer type: " + tag)
		}
		if err := codec.Writer(writer, fieldValue); err != nil {
			return err
		}
	}
	return nil
}

func DeserializePacket(reader ByteArrayReader, packetType reflect.Type) (reflect.Value, error) {
	packetPtr := reflect.New(packetType)
	packetValue := packetPtr.Elem()
	for i := 0; i < packetType.NumField(); i++ {
		// Get the field tag value
		field := packetType.Field(i)
		tag := field.Tag.Get("packet")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		fieldValue := packetValue.Field(i)
		codec, ok := typeCodecs[tag]
		if !ok {
			return packetPtr, errors.New("unknown serializer type: " + tag)
		}
		if err := codec.Reader(reader, fieldValue); err != nil {
			return packetPtr, err
		}
	}
	return packetPtr, nil
}

func bakeWriter[T any](writeFn func(ByteArrayWriter, T) error, getterFn func(reflect.Value) T) func(ByteArrayWriter, reflect.Value) error {
	return func(writer ByteArrayWriter, value reflect.Value) error {
		return writeFn(writer, getterFn(value))
	}
}

func bakeReader[T any](readFn func(ByteArrayReader) (T, error), setterFn func(reflect.Value, T)) func(ByteArrayReader, reflect.Value) error {
	return func(reader ByteArrayReader, value reflect.Value) error {
		val, err := readFn(reader)
		if err != nil {
			return err
		}
		setterFn(value, val)
		return nil
	}
}

func bakeIntWriter[T constraints.Signed](writeFn func(ByteArrayWriter, T) error) func(ByteArrayWriter, reflect.Value) error {
	return bakeWriter(writeFn, func(value reflect.Value) T {
		return T(value.Int())
	})
}

func bakeIntReader[T constraints.Signed](readFn func(ByteArrayReader) (T, error)) func(ByteArrayReader, reflect.Value) error {
	return bakeReader(readFn, func(value reflect.Value, val T) {
		value.SetInt(int64(val))
	})
}

func bakeIntCodec[T constraints.Signed](writeFn func(ByteArrayWriter, T) error, readFn func(ByteArrayReader) (T, error)) TypeCodec {
	return TypeCodec{
		Writer: bakeIntWriter(writeFn),
		Reader: bakeIntReader(readFn),
	}
}

func init() {
	RegisterCodec("byte", TypeCodec{
		Writer: bakeWriter(writeByte, func(value reflect.Value) byte {
			return byte(value.Uint())
		}),
		Reader: bakeReader(readByte, func(value reflect.Value, val byte) {
			value.SetUint(uint64(val))
		}),
	})

	RegisterCodec("int8", bakeIntCodec(writeInt8, readInt8))
	RegisterCodec("int16", bakeIntCodec(writeInt16, readInt16))
	RegisterCodec("int", bakeIntCodec(writeInt, readInt))
	RegisterCodec("int64", bakeIntCodec(writeInt64, readInt64))
	RegisterCodec("varint", bakeIntCodec(writeVarInt, readVarInt))

	RegisterCodec("float", TypeCodec{
		Writer: bakeWriter(writeFloat, func(value reflect.Value) float32 {
			return float32(value.Float())
		}),
		Reader: bakeReader(readFloat, func(value reflect.Value, val float32) {
			value.SetFloat(float64(val))
		}),
	})
	RegisterCodec("float64", TypeCodec{
		Writer: bakeWriter(writeFloat64, reflect.Value.Float),
		Reader: bakeReader(readFloat64, reflect.Value.SetFloat),
	})
	RegisterCodec("bool", TypeCodec{
		Writer: bakeWriter(writeBool, reflect.Value.Bool),
		Reader: bakeReader(readBool, reflect.Value.SetBool),
	})
	RegisterCodec("string", TypeCodec{
		Writer: bakeWriter(writeString, reflect.Value.String),
		Reader: bakeReader(readString, reflect.Value.SetString),
	})
	RegisterCodec("byte_array", TypeCodec{
		Writer: bakeWriter(writeByteArray, reflect.Value.Bytes),
		Reader: bakeReader(readByteArray, reflect.Value.SetBytes),
	})
	RegisterCodec("component", TypeCodec{
		Writer: bakeWriter(writeChatComponent, func(value reflect.Value) ChatComponent {
			return value.Interface().(ChatComponent)
		}),
		Reader: bakeReader(readChatComponent, func(value reflect.Value, val ChatComponent) {
			value.Set(reflect.ValueOf(val))
		}),
	})
	RegisterCodec("json", TypeCodec{
		Writer: bakeWriter(writeJson, reflect.Value.Interface),
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
