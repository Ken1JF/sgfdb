package sgfdb_test

import (
	"fmt"
	"github.com/Ken1JF/sgf"
	. "github.com/Ken1JF/sgfdb"
	"os"
	"runtime"
	//	"testing"
)

const gogod_dir = "/usr/local/GoGoD"
const SGFSpecFile = "../sgf/sgf_properties_spec.txt"

func ExampleSgfDbTypeSizes() {
	PrintSgfDbTypeSizes()
	// Output:
	// Type TraceRec size 40 alignment 8
	// Type CountDirRequest size 88 alignment 8
}

func ExampleReadWriteDatabase() {
	//	if testing.Short() {
	//		t.Skip("skipping test in short mode.")
	//	}
	//finfo
	_, err := os.Stat(gogod_dir)
	if err != nil {
		fmt.Println("skipping test, error:", err, "accessing:", gogod_dir)
		//		t.Skip("skipping test, error:", err, "accessing:", gogod_dir)
		return
	}

	fmt.Println("running test, OK accessing:", gogod_dir)

	// Need to run this with 1 CPU to get reproducible results
	oldMaxProcs := 0
	nCPUsOnMachine := runtime.NumCPU()
	if nCPUsOnMachine > 1 {
		oldMaxProcs = runtime.GOMAXPROCS(1)
		fmt.Printf("num CPUs on Machine = %d, default max Procs was %d, now set to 1\n", nCPUsOnMachine, oldMaxProcs)
	} else {
		fmt.Printf("num CPUs on Machine = %d.\n", nCPUsOnMachine)
	}

	// do not ask for verification of SGF Specification file,
	// or ask for verbose output. These are done in sgf_test.go
	// If that test is ok, then the file is ok.
	errN := sgf.SetupSGFProperties(SGFSpecFile, false, false)
	if errN == 0 {
		CountFilesAndMoves(gogod_dir+"/Go/Database/", 0, false)
	}

	// if GOMAXPROCS was changed, set it back
	if oldMaxProcs != 0 {
		fmt.Printf(" max Procs set back to %d.\n", oldMaxProcs)
		runtime.GOMAXPROCS(oldMaxProcs)
	}
	// Output:
	// running test, OK accessing: /usr/local/GoGoD
	// num CPUs on Machine = 4, default max Procs was 1, now set to 1
	//   0:0196-1699, files: 784, moves: 149431
	//   1:1700-99, files: 1195, moves: 248166
	//   2:1800-49, files: 1254, moves: 232289
	//   3:1850-99, files: 1618, moves: 290802
	//   4:1900-09, files: 412, moves: 81389
	//   5:1910-19, files: 379, moves: 73195
	//   6:1920-29, files: 895, moves: 178732
	//   7:1930-39, files: 2224, moves: 469415
	//   8:1940-49, files: 743, moves: 155292
	//   9:1950-59, files: 1636, moves: 343773
	//  10:1960-69, files: 3178, moves: 647853
	//  11:1970-75, files: 2201, moves: 442334
	//  12:1976-79, files: 2126, moves: 429100
	//  13:1980, files: 804, moves: 156099
	//  14:1981, files: 651, moves: 129838
	//  15:1982, files: 718, moves: 140165
	//  16:1983, files: 862, moves: 174136
	//  17:1984, files: 744, moves: 154795
	//  18:1985, files: 978, moves: 202844
	//  19:1986, files: 1006, moves: 211331
	//  20:1987, files: 963, moves: 194109
	//  21:1988, files: 1144, moves: 236411
	//  22:1989, files: 1414, moves: 288660
	//  23:1990, files: 1377, moves: 283506
	//  24:1991, files: 1279, moves: 265952
	//  25:1992, files: 1374, moves: 286559
	//  26:1993, files: 1383, moves: 294227
	//  27:1994, files: 1326, moves: 278442
	//  28:1995, files: 1620, moves: 343337
	//  29:1996, files: 1873, moves: 393597
	//  30:1997, files: 1691, moves: 353836
	//  31:1998, files: 1734, moves: 361426
	//  32:1999, files: 1407, moves: 293231
	//  33:2000, files: 1659, moves: 343388
	//  34:2001, files: 2007, moves: 421291
	//  35:2002, files: 1728, moves: 363739
	//  36:2003, files: 2013, moves: 429808
	//  37:2004, files: 1949, moves: 414287
	//  38:2005, files: 1875, moves: 401196
	//  39:2006, files: 2778, moves: 598914
	//  40:2007, files: 2716, moves: 578552
	//  41:2008, files: 2985, moves: 641211
	//  42:2009, files: 2622, moves: 563081
	//  43:2010, files: 2635, moves: 569596
	//  44:2011, files: 2219, moves: 473040
	//  45:2012, files: 0, moves: 0
	//  47:GoLibrary2, files: 0, moves: 0
	//  50:Onomasticon2, files: 0, moves: 0
	// Total SGF files = 70179, total moves = 14582375
	//  max Procs set back to 1.
}
