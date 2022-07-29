package db

import (
	"testing"
)

func TestConn(t *testing.T) {
	t.Log(PDB.Error)
	t.FailNow()
}
