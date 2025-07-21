package importer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GetBinarySectionFromBin parses the bin (pbl) file and extracts the binary string of a OLE PowerBuilder Binary
// Data Section. The prefixes (1 character X before some.bin, and before each line (VW,ST)) are arbitrary
// chosen and will be corrected by a full-build.
//
// Start of PowerBuilder Binary Data Section : Do NOT Edit
// 0Xsome.bin
// VW<binaryString-3992chars>
// ST<binaryString-3992chars>
// 1Xsome.bin
// End of PowerBuilder Binary Data Section : No Source Expected After This Point
func GetBinarySectionFromBin(binFile string) ([]byte, error) {
	var binarySection = make([]byte, 0)
	binarySection = append(binarySection, []byte("Start of PowerBuilder Binary Data Section : Do NOT Edit\r\n")...)
	binarySection = append(binarySection, []byte(fmt.Sprintf("0A%s\r\n", filepath.Base(binFile)))...)
	hexString, err := sortBinBytesHex(binFile)
	if err != nil {
		return nil, err
	}
	blockSize := 3992
	remainder := len(hexString) % blockSize
	if remainder != 0 {
		zerosToAdd := blockSize - remainder
		hexString += strings.Repeat("0", zerosToAdd)
	}

	for i := 0; i < len(hexString)/blockSize; i++ {
		// 2A is arbitrary and wrong: after full-build, refresh and export the right magic numbers appear whereas
		// the byte string stays the same.
		firstIndex := i * blockSize
		binarySection = append(binarySection, []byte(fmt.Sprintf("2A%s\r\n",
			hexString[firstIndex:firstIndex+blockSize]))...)
	}

	binarySection = append(binarySection, []byte(fmt.Sprintf("1A%s\r\n", filepath.Base(binFile)))...)
	binarySection = append(binarySection,
		[]byte("End of PowerBuilder Binary Data Section : No Source Expected After This Point")...)
	return binarySection, nil
}

// sortBinBytesHex reads a bin files blocks according to scheme below and
// returns a related hex string used for binary section. Each sequence of 4 bytes are arranged newly:
// e.g. 01 02 03 04 -> 04 03 02 01
// +--------------------------------------------------------------+
// I Data Block (512 Byte)                                        I
// +-----------+------------+-------------------------------------+
// I Pos.      I Type       I Information                         I
// +-----------+------------+-------------------------------------+
// I   1 - 4   I Char(4)    I 'DAT*'                              I
// I   5 - 8   I Long       I Offset of next data block or 0      I
// I   9 - 10  I Integer    I Length of data in block             I
// I  11 - XXX I Blob{}     I Data (maximum Length is 502         I
// +-----------+------------+-------------------------------------+
func sortBinBytesHex(binFile string) (string, error) {
	file, err := os.Open(binFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	var rawBytes = make([]byte, fileInfo.Size())
	_, err = file.Read(rawBytes)
	if err != nil {
		return "", err
	}
	builder := &strings.Builder{}

	// find first Dat* block
	numberOfBlocks := bytes.Count(rawBytes, []byte("DAT*"))
	index := bytes.Index(rawBytes, []byte("DAT*"))
	// read one data blob, always 4 hex values, then revert order of those: first one is last one and last one is first
	// one, e.g. D0CF11E0 -> E011CFD0
	var lastRemainder []byte
	part, blockLength := readBlock(file, rawBytes, int64(index), &lastRemainder)
	builder.WriteString(part)
	for i := 1; i < numberOfBlocks; i++ {
		index = index + int(blockLength) + 10 // + from last block
		part, blockLength = readBlock(file, rawBytes, int64(index), &lastRemainder)
		builder.WriteString(part)
	}
	return strings.ToLower(builder.String()), nil
}

// remainder is overhead of last block since 502 can not be divided by 4. Returns length of read block and remainder
func readBlock(file *os.File, rawBytes []byte, startIndex int64, lastRemainder *[]byte) (string, uint16) {
	builder := strings.Builder{}
	// check if really new DAT* data block
	if !bytes.Equal(rawBytes[startIndex:startIndex+4], []byte{68, 65, 84, 42}) {
		log.Fatal("no DAT* block for current start index")
	}
	// startIndex = DAT*
	// +10 where data block starts
	_, err := file.Seek(startIndex+int64(10), io.SeekStart)
	if err != nil {
		log.Fatal(err)
	}
	blockLength := binary.LittleEndian.Uint16(rawBytes[startIndex+8 : startIndex+10])
	buf := make([]byte, 4)
	bytesRead := int64(0)

	if lastRemainder != nil && len(*lastRemainder) == 2 {
		buf[0] = (*lastRemainder)[0]
		buf[1] = (*lastRemainder)[1]
		bufNext := make([]byte, 2)
		n, err := file.Read(bufNext)
		if n != 2 || err != nil {
			log.Fatal(err)
		}
		buf[2], buf[3] = bufNext[0], bufNext[1]
		processBuffer(buf, &builder)
		bytesRead += 2
	}

	for uint16(bytesRead) < blockLength {
		n, err := file.Read(buf)
		if n == 0 || err == io.EOF {
			break
		}
		if uint16(bytesRead)+4 > blockLength {
			if (blockLength - uint16(bytesRead)) != 2 {
				log.Fatal("remainder length of block length 502 can only be 0 or 2")
			}
			// length 502 can not be divided by 4 -> last 2 hex values must be added to next block.
			(*lastRemainder) = append((*lastRemainder), buf[0])
			(*lastRemainder) = append((*lastRemainder), buf[1])
			return builder.String(), blockLength
		}
		processBuffer(buf, &builder)
		bytesRead += 4
	}
	*lastRemainder = []byte{}
	return builder.String(), blockLength
}

// processBuffer swaps the first 4 buffer bytes 0 -> 3, ..., 3 -> 0 and adds them in the new order to builder.
func processBuffer(buf []byte, builder *strings.Builder) {
	// Reverse the 4 bytes in place
	for i, j := 0, 3; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	builder.WriteString(fmt.Sprintf("%02X%02X%02X%02X", buf[0], buf[1], buf[2], buf[3]))
}
