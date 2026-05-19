// Package gamedata decrypts and decompresses Albion's GameData .bin files
// and converts the embedded XML into a more agent-friendly form.
//
// The on-disk format is DES-CBC(Gzip(XML)) with a fixed 8-byte key/IV that
// every third-party Albion tool uses. We mirror SAT's
// StatisticAnalysisTool.Extractor here.
package gamedata

import (
	"bytes"
	"compress/gzip"
	"crypto/cipher"
	"crypto/des"
	"fmt"
	"io"
	"os"
)

// Well-known DES key/IV used to encrypt every Albion GameData .bin file.
// Identical to BinaryDecrypter.cs in SAT and to every other third-party
// extractor in the wild.
var (
	desKey = []byte{48, 239, 114, 71, 66, 242, 4, 50}
	desIV  = []byte{14, 166, 220, 137, 219, 237, 220, 79}
)

// DecryptAndDecompress reads a .bin file, undoes DES-CBC + gzip, and returns
// the embedded XML bytes.
func DecryptAndDecompress(binPath string) ([]byte, error) {
	encrypted, err := os.ReadFile(binPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", binPath, err)
	}
	if len(encrypted)%des.BlockSize != 0 {
		return nil, fmt.Errorf("%s: length %d is not a multiple of DES block size", binPath, len(encrypted))
	}

	block, err := des.NewCipher(desKey)
	if err != nil {
		return nil, fmt.Errorf("init DES cipher: %w", err)
	}
	dec := cipher.NewCBCDecrypter(block, desIV)

	plain := make([]byte, len(encrypted))
	dec.CryptBlocks(plain, encrypted)

	// Strip PKCS#5/7 padding.
	if n := len(plain); n > 0 {
		pad := int(plain[n-1])
		if pad > 0 && pad <= des.BlockSize && pad <= n {
			plain = plain[:n-pad]
		}
	}

	gz, err := gzip.NewReader(bytes.NewReader(plain))
	if err != nil {
		return nil, fmt.Errorf("gzip header: %w", err)
	}
	defer gz.Close()

	out, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("gzip body: %w", err)
	}
	return out, nil
}
