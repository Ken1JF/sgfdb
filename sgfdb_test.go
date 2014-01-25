package sgfdb_test

import (
	"gitHub.com/Ken1JF/ahgo/sgf"
	. "gitHub.com/Ken1JF/ahgo/sgfdb"
	"os"
	"testing"
)

const gogod_dir = "/usr/local/GoGoD"
const SGFSpecFile = "../sgf_properties_spec.txt"

func ExampleSgfDbTypeSizes() {
	PrintSgfDbTypeSizes()
	// Output:
	// Type TraceRec size 40 alignment 8
	// Type CountDirRequest size 88 alignment 8
}

func TestReadWriteDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	//finfo
	_, err := os.Stat(gogod_dir)
	if err != nil {
		t.Skip("skipping test, error:", err, "accessing:", gogod_dir)
	}

	// do not ask for verification of SGF Specification file,
	// or ask for verbose output. These are done in sgf_test.go
	// If that test is ok, then the file is ok.
	errN := sgf.SetupSGFProperties(SGFSpecFile, false, false)
	if errN == 0 {
		CountFilesAndMoves(gogod_dir+"/Go/Database/", 0)
	}
}
