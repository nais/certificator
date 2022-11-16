package filereader_test

import (
	"os"
	"testing"

	"github.com/nais/certificator/pkg/filereader"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	f, err := os.Open("../../testdata/cacert.pem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	certs, err := filereader.Read(f)
	assert.NoError(t, err)

	for _, cert := range certs {
		t.Logf("%s %#v %x", cert.Type, cert.Headers, cert.Bytes[:8])
	}
}
