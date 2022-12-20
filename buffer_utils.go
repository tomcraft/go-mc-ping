package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
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

func readVarInt(reader io.ByteReader) (int, error) {
	value, err := binary.ReadUvarint(reader)
	return int(value), err
}

func writeVarInt(writer io.ByteWriter, value int) error {
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

func serializePacket(writer ByteArrayWriter, packet any) error {
	// ValueOf returns a Value representing the run-time data
	v := reflect.ValueOf(packet)
	vType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		tag := vType.Field(i).Tag.Get("packet")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		vValue := v.Field(i)
		var err error
		switch tag {
		case "int8":
			err = writeInt8(writer, int8(vValue.Int()))
		case "byte":
			err = writeByte(writer, byte(vValue.Uint()))
		case "varint":
			err = writeVarInt(writer, int(vValue.Int()))
		case "int":
			err = writeInt(writer, int(vValue.Int()))
		case "int16":
			err = writeInt16(writer, int16(vValue.Int()))
		case "int64":
			err = writeInt64(writer, vValue.Int())
		case "float":
			err = writeFloat(writer, float32(vValue.Float()))
		case "float64":
			err = writeFloat64(writer, vValue.Float())
		case "string":
			err = writeString(writer, vValue.String())
		case "byte_array":
			err = writeByteArray(writer, vValue.Bytes())
		case "bool":
			err = writeBool(writer, vValue.Bool())
		case "component":
			err = writeChatComponent(writer, vValue.Interface().(ChatComponent))
		case "json":
			err = writeJson(writer, vValue.Interface())
		default:
			err = errors.New("unknown serializer type: " + tag)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func deserializePacket(reader ByteArrayReader, vType reflect.Type) (reflect.Value, error) {
	packet := reflect.New(vType)
	// ValueOf returns a Value representing the run-time data
	for i := 0; i < vType.NumField(); i++ {
		// Get the field tag value
		tag := vType.Field(i).Tag.Get("packet")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		vValue := packet.Elem().Field(i)
		var err error
		var val any
		switch tag {
		case "int8":
			val, err = readInt8(reader)
			vValue.SetInt(int64(val.(int8)))
		case "byte":
			val, err = readByte(reader)
			vValue.SetUint(uint64(val.(byte)))
		case "varint":
			val, err = readVarInt(reader)
			vValue.SetInt(int64(val.(int)))
		case "int":
			val, err = readInt(reader)
			vValue.SetInt(int64(val.(int)))
		case "int16":
			val, err = readInt16(reader)
			vValue.SetInt(int64(val.(int16)))
		case "int64":
			val, err = readInt64(reader)
			vValue.SetInt(val.(int64))
		case "float":
			val, err = readFloat(reader)
			vValue.SetFloat(float64(val.(float32)))
		case "float64":
			val, err = readFloat64(reader)
			vValue.SetFloat(val.(float64))
		case "string":
			val, err = readString(reader)
			vValue.SetString(val.(string))
		case "byte_array":
			val, err = readByteArray(reader)
			vValue.SetBytes(val.([]byte))
		case "bool":
			val, err = readBool(reader)
			vValue.SetBool(val.(bool))
		case "component":
			val, err = readChatComponent(reader)
			vValue.Set(reflect.ValueOf(val))
		case "json":
			reflectVal := reflect.New(vValue.Type())
			val, err = readJson(reader, reflectVal)
			vValue.Set(reflectVal)
		default:
			err = errors.New("unknown serializer type: " + tag)
		}
		if err != nil {
			return packet, err
		}
	}
	return packet, nil
}

func wrapHandler[T any](mappedFunction func(client *Client, packet T) error) PacketHandler {
	t := reflect.TypeOf(mappedFunction).In(1)
	return func(client *Client, reader ByteArrayReader) error {
		pp, err := deserializePacket(reader, t)
		if err != nil {
			return err
		}
		return mappedFunction(client, *(pp.Interface().(*T)))
	}
}
