package confile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	confile := "./conf.yaml"
	err := NewConfigfile().Parse(confile, "", 0)
	assert.NoError(t, err)
}
