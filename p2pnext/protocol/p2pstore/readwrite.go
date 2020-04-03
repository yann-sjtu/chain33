package p2pstore

import (
	"bufio"

	"github.com/33cn/chain33/types"
)

func readMessage(reader *bufio.Reader, msg types.Message) error {
	var data []byte
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		data = append(data, buf[:n]...)
		if n < 1024 {
			break
		}
	}
	return types.Decode(data, msg)
}

func writeMessage(writer *bufio.Writer, msg types.Message) error {
	b := types.Encode(msg)
	_, err := writer.Write(b)
	if err != nil {
		return err
	}
	return writer.Flush()
}
