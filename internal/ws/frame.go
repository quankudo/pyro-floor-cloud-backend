package ws

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// ReadFrame đọc một frame WebSocket từ conn, xử lý text, ping, pong, close
func ReadFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	opcode := header[0] & 0x0F
	fin := header[0]&0x80 != 0

	if !fin {
		return nil, fmt.Errorf("fragmented frames not supported")
	}

	mask := header[1]&0x80 != 0
	payloadLen := int(header[1] & 0x7F)

	// đọc payload length extended nếu > 125
	// determine payload length

	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint64(ext))
	default:
		// payloadLen < 126, no extra bytes
	}

	// đọc mask key nếu có
	var maskKey []byte
	if mask {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(conn, maskKey); err != nil {
			return nil, err
		}
	}

	// đọc payload
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	// unmask nếu cần
	if mask {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	// xử lý opcode
	switch opcode {
	case 0x1: // text
		return payload, nil
	case 0x8: // close
		return nil, io.EOF
	case 0x9: // ping
		WritePong(conn)
		return nil, nil
	case 0xA: // pong
		// ignore pong
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported opcode %d", opcode)
	}
}

// WriteFrame gửi text frame
func WriteFrame(conn net.Conn, data []byte) error {
	frame := []byte{0x81} // FIN + text frame

	length := len(data)
	switch {
	case length < 126:
		frame = append(frame, byte(length))
	case length <= 65535:
		frame = append(frame, 126)
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		frame = append(frame, lenBytes...)
	default:
		frame = append(frame, 127)
		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, uint64(length))
		frame = append(frame, lenBytes...)
	}

	frame = append(frame, data...)

	_, err := conn.Write(frame)
	return err
}

// WritePong gửi pong frame
func WritePong(conn net.Conn) {
	frame := []byte{0x8A, 0x00} // FIN + pong, payload 0
	_, _ = conn.Write(frame)
}
