package sub

// Reed Solomon 9/3 forward error correction, intended to be sent as 9 pieces where 3 uncorrupted parts allows assembly of the message
import (
	"encoding/binary"
	"github.com/vivint/infectious"
	"hash/crc32"
)

var (
	rsTotal    = 9
	rsRequired = 3
	rsFEC      = func() *infectious.FEC {
		fec, err := infectious.NewFEC(3, 9)
		check(err, "creating 3,9 FEC codec", true)
		return fec
	}()
)

// padData appends a 2 byte length prefix, and pads to a multiple of rsTotal. An empty slice will be returned if the total length is greater than maxMessageSize.
func padData(data []byte) (out []byte) {
	dataLen := len(data)
	prefixBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(prefixBytes, uint16(dataLen))
	data = append(prefixBytes, data...)
	dataLen = len(data)
	if dataLen > maxMessageSize {
		return []byte{}
	}
	chunkLen := (dataLen) / rsTotal
	chunkMod := (dataLen) % rsTotal
	if chunkMod != 0 {
		chunkLen++
	}
	padLen := rsTotal*chunkLen - dataLen
	out = append(data, make([]byte, padLen)...)
	return
}

func rsEncode(data []byte) (chunks [][]byte) {
	// First we must pad the data
	data = padData(data)
	shares := make([]infectious.Share, rsTotal)
	output := func(s infectious.Share) {
		shares[s.Number] = s.DeepCopy()
	}
	err := rsFEC.Encode(data, output)
	check(err, "sub.rsEncode", false)
	for i := range shares {
		// Append the chunk number to the front of the chunk
		chunk := append([]byte{byte(shares[i].Number)}, shares[i].Data...)
		// Checksum includes chunk number byte so we know if its checksum is incorrect so could the chunk number be
		checksum := crc32.Checksum(chunk, crc32.MakeTable(crc32.Castagnoli))
		checkbytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(checkbytes, checksum)
		chunk = append(chunk, checkbytes...)
		chunks = append(chunks, chunk)
	}
	return
}

func rsDecode(chunks [][]byte) (data []byte) {
	var shares []infectious.Share
	for i := range chunks {
		// First get the crc32 checksum suffix and convert to uint32
		bodyLen := len(chunks[i])
		body := chunks[i][:bodyLen-4]
		check := chunks[i][bodyLen-4:]
		checkSum := binary.LittleEndian.Uint32(check)
		checksum := crc32.Checksum(body, crc32.MakeTable(crc32.Castagnoli))
		if checkSum == checksum {
			// The chunk appears to be complete, so assemble the infectious.Share struct
			share := infectious.Share{
				Number: int(body[0]),
				Data:   body[1:],
			}
			shares = append(shares, share)
		}
	}
	data, err := rsFEC.Decode(nil, shares)
	check(err, "sub.rsDecode", false)
	return
}
