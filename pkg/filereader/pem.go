package filereader

import (
	"bytes"
	"encoding/pem"
	"io"
)

func Read(r io.Reader) ([]*pem.Block, error) {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	if err != nil {
		return nil, err
	}

	data := buf.Bytes()
	certs := make([]*pem.Block, 0)
	for {
		var cert *pem.Block
		cert, data = pem.Decode(data)
		if cert == nil {
			break
		} else {
			certs = append(certs, cert)
		}
	}

	return certs, nil
}
