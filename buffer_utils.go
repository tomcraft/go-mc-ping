package main

import (
	"encoding/binary"
	"io"
	"math"
)

type ByteArrayReader interface {
	io.Reader
	io.ByteReader
}

type ByteArrayWriter interface {
	io.Writer
	io.ByteWriter
}

func writeByte(writer ByteArrayWriter, value int8) error {
	return writer.WriteByte(byte(value))
}

func writeUnsignedByte(writer ByteArrayWriter, value byte) error {
	return writer.WriteByte(value)
}

func readVarInt(reader io.ByteReader) (int, error) {
	varint, err := binary.ReadUvarint(reader)
	return int(varint), err
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

func readShort(reader ByteArrayReader) (int16, error) {
	array := make([]byte, 2)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(array)), nil
}

func readLong(reader ByteArrayReader) (int64, error) {
	array := make([]byte, 8)
	if _, err := reader.Read(array); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(array)), nil
}

func writeLong(writer ByteArrayWriter, value int64) error {
	array := make([]byte, 8)
	binary.BigEndian.PutUint64(array, uint64(value))
	_, err := writer.Write(array)
	return err
}

func writeFloat(writer ByteArrayWriter, value float32) error {
	array := make([]byte, 4)
	binary.BigEndian.PutUint32(array, math.Float32bits(value))
	_, err := writer.Write(array)
	return err
}

func writeDouble(writer ByteArrayWriter, value float64) error {
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

func writeBool(writer ByteArrayWriter, value bool) error {
	if value {
		return writer.WriteByte(1)
	} else {
		return writer.WriteByte(0)
	}
}
