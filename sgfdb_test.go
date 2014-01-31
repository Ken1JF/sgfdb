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
const second_dir = "../"
const SGFSpecFile = "../sgf/sgf_properties_spec.txt"

func ExampleSgfDbTypeSizes() {
	PrintSgfDbTypeSizes()
	// Output:
	// Type TraceRec size 40 alignment 8
	// Type CountDirRequest size 96 alignment 8
}

// Expected output when the link in /usr/local is in place: GoGoD -> /Users/ken/Documents/GO/GoGoD
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
//  46:GoLibrary2, files: 0, moves: 0
//  47:Onomasticon2, files: 0, moves: 0
// Total SGF files = 70179, total moves = 14582375
//  max Procs set back to 1.

func ExampleReadDatabaseCountMoves() {
	var useDir = ""
	var usingSecondDir = false
	var linkTestOut = false

	_, err := os.Stat(gogod_dir)
	if err != nil {
		fmt.Println("Not doing GoGoD ExampleReadDatabaseCountMoves:", err, "accessing:", gogod_dir)
		_, err2 := os.Stat(second_dir)
		if err2 != nil {
			fmt.Println("ExampleReadDatabaseCountMoves Error:", err2, "no secondary directory:", second_dir)
			return
		}
		useDir = second_dir + "sgf/"
		usingSecondDir = true
	} else {
		useDir = gogod_dir + "Go/Database/"
	}

	// If doing second_dir, check if sgf/testout exists.
	// If not link it, temporarily.
	if usingSecondDir {
		_, err = os.Stat(useDir + "testout")
		if err != nil {
			err2 := os.Symlink(useDir+"testdata", useDir+"testout")
			if err2 != nil {
				fmt.Printf("ExampleReadDatabaseCountMoves Error: %s trying to link: %s\n", err2, useDir+"testout")
			} else {
				// debug: fmt.Printf("Set Symblink to:%s\n", useDir + "testout")
				linkTestOut = true
			}
		}
	} else {
		fmt.Printf("ExampleReadDatabaseCountMoves: not using secondary dir.\n")
	}

	fmt.Println("running ExampleReadDatabaseCountMoves, OK accessing:", useDir)

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
		CountFilesAndMoves(useDir, 0, false, sgf.ParserIgnoreUnknSGF)
	}

	// If testout was linked, unlink it.
	if linkTestOut {
		err = os.Remove(useDir + "testout")
		if err != nil {
			fmt.Printf("ExampleReadDatabaseCountMoves, Error: %s attempting to remove symbolic link to:%s\n", err, useDir+"testout")
		}
	}
	// if GOMAXPROCS was changed, set it back
	if oldMaxProcs != 0 {
		fmt.Printf(" max Procs set back to %d.\n", oldMaxProcs)
		runtime.GOMAXPROCS(oldMaxProcs)
	}
	// Output:
	// Not doing GoGoD ExampleReadDatabaseCountMoves: stat /usr/local/GoGoD: no such file or directory accessing: /usr/local/GoGoD
	// running ExampleReadDatabaseCountMoves, OK accessing: ../sgf/
	// num CPUs on Machine = 4, default max Procs was 1, now set to 1
	//   0:testdata, files: 55, moves: 9127
	//   1:testout, files: 55, moves: 9127
	// Total SGF files = 110, total moves = 18254
	//  max Procs set back to 1.
}

// ExampleReadWriteDatabase will use the GoGoD database if present.
// If not, it will use the directories found in ../sgf/.
func ExampleReadWriteDatabase() {
	var useDir = ""
	var testOutDir = ""
	var usingSecondDir = false
	var linkTestOut = false

	// Check if GoGoD is present
	_, err := os.Stat(gogod_dir)
	if err != nil {
		fmt.Println("Not doing GoGoD ExampleReadWriteDatabase:", err, "accessing:", gogod_dir)
		_, err2 := os.Stat(second_dir)
		if err2 != nil {
			fmt.Println("ExampleReadWriteDatabase, Error:", err2, "no secondary directory:", second_dir)
			return
		}
		useDir = second_dir + "sgf/"
		testOutDir = second_dir + "sgfdb/dbout/"
		usingSecondDir = true
	} else {
		useDir = gogod_dir + "Go/Database/"
		testOutDir = gogod_dir + "dbout/"
	}
	// Check that the input directory exists.
	_, err = os.Stat(useDir)
	if err != nil {
		fmt.Println("Error: ExampleReadWriteDatabase:", err, "accessing root directory:", useDir)
		return
	}
	// Check the output directory. If missing, create it.
	_, err = os.Stat(testOutDir)
	if err != nil {
		err2 := os.MkdirAll(testOutDir, os.ModeDir|os.ModePerm)
		if err2 != nil {
			fmt.Println("Error: ExampleReadWriteDatabase:", err2, "trying to create test output directory:", testOutDir)
			fmt.Println("Original Error:", err, "trying os.Stat")
			return
		}
	}

	// If doing second_dir, check if sgf/testout exists.
	// If not link it, temporarily.
	if usingSecondDir {
		// debug: finfo, err := os.Stat(useDir + "testout")
		_, err := os.Stat(useDir + "testout")
		if err != nil {
			err2 := os.Symlink(useDir+"testdata", useDir+"testout")
			if err2 != nil {
				fmt.Printf("ExampleReadWriteDatabase Error: %s trying to link: %s\n", err2, useDir+"testout")
			} else {
				// debug: fmt.Printf("Set Symblink to:%s\n", useDir + "testout")
				linkTestOut = true
			}
		} else {
			// debug: fmt.Printf("Stat of %s = %v\n", useDir + "testout", finfo)
		}
	} else {
		fmt.Printf("ExampleReadDatabaseCountMoves: not using secondary dir.\n")
	}

	fmt.Println("Running ExampleReadWriteDatabase, OK using:", useDir, "with output to:", testOutDir)

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
		CountFilesAndMoves(useDir, 0, false, sgf.ParserIgnoreUnknSGF)
		skipFiles := 0 // do we need an option to set this???
		stat := ReadAndWriteDatabase(useDir, testOutDir, 0, 0, skipFiles, sgf.ParserIgnoreUnknSGF)
		if stat > 0 {
			fmt.Printf("ExampleReadWriteDatabase Errors during reading and writing database: %d\n", stat)
		}
		sgf.ReportSGFCounts()
	}

	// If testout was linked, unlink it.
	if linkTestOut {
		err = os.Remove(useDir + "testout")
		if err != nil {
			fmt.Printf("ExampleReadWriteDatabase Error: %s attempting to remove symbolic link to:%s\n", err, useDir+"testout")
		}
	}

	// if GOMAXPROCS was changed, set it back
	if oldMaxProcs != 0 {
		fmt.Printf(" max Procs set back to %d.\n", oldMaxProcs)
		runtime.GOMAXPROCS(oldMaxProcs)
	}
	// Output:
	// Not doing GoGoD ExampleReadWriteDatabase: stat /usr/local/GoGoD: no such file or directory accessing: /usr/local/GoGoD
	// Running ExampleReadWriteDatabase, OK using: ../sgf/ with output to: ../sgfdb/dbout/
	// num CPUs on Machine = 4, default max Procs was 1, now set to 1
	//   0:testdata, files: 55, moves: 9127
	//   1:testout, files: 55, moves: 9127
	// Total SGF files = 110, total moves = 18254
	// Reading and writing database, db_dir = ../sgf/, testout_dir = ../sgfdb/dbout/
	//   0:testdata, files: 55, tokens: 0
	//   1:testout, files: 55, tokens: 0
	// Total SGF files = 110, tokens = 0
	// Property AB used 16 times.
	// Property AP used 18 times.
	// Property B used 9344 times.
	// Property BL used 1908 times.
	// Property BR used 72 times.
	// Property BT used 6 times.
	// Property C used 832 times.
	// Property CA used 70 times.
	// Property DT used 72 times.
	// Property EV used 56 times.
	// Property FF used 96 times.
	// Property FG used 38 times.
	// Property GM used 94 times.
	// Property HA used 16 times.
	// Property KM used 74 times.
	// Property LB used 18 times.
	// Property MN used 24 times.
	// Property N used 54 times.
	// Property PB used 72 times.
	// Property PC used 46 times.
	// Property PM used 4 times.
	// Property PW used 72 times.
	// Property RE used 70 times.
	// Property RO used 44 times.
	// Property RU used 42 times.
	// Property SO used 4 times.
	// Property ST used 18 times.
	// Property SZ used 108 times.
	// Property TB used 10 times.
	// Property TM used 18 times.
	// Property TR used 6 times.
	// Property TW used 10 times.
	// Property US used 4 times.
	// Property VW used 18 times.
	// Property W used 8996 times.
	// Property WL used 1916 times.
	// Property WR used 72 times.
	// Property WT used 6 times.
	// Property ?Unkn? used 2 times.
	// FirstBRankNotSet 46 times.
	// FirstWRankNotSet 69 times.
	// Handicap 9 occurred 10 times.
	// Handicap 8 occurred 4 times.
	// Handicap 7 occurred 2 times.
	// Total Handicap games 16 with 3 different Handicaps
	// Total Old Handicap games 0 with 0 different Old Handicaps
	// Result W+R occurred 20 times.
	// Result B+R occurred 14 times.
	// Result W+Resign occurred 6 times.
	// Result W+ occurred 4 times.
	// Result B+ occurred 2 times.
	// Result B+0.75 occurred 2 times.
	// Result B+1.75 occurred 2 times.
	// Result B+7.5 occurred 2 times.
	// Result W+1.5 occurred 2 times.
	// Result W+13.50 occurred 2 times.
	// Result W+2.25 occurred 2 times.
	// Result W+2.5 occurred 2 times.
	// Result W+29.50 occurred 2 times.
	// Result W+35.50 occurred 2 times.
	// Result W+4.5 occurred 2 times.
	// Result W+5.50 occurred 2 times.
	// Result W+8.50 occurred 2 times.
	// Total Result games 70 with 17 different Results
	// Result comment moves beyond % not known occurred 4 times.
	// Result comment Moves beyond % not known occurred 2 times.
	// Total Result comment games 6 with 2 different Result comments
	// Rules Chinese occurred 18 times.
	// Rules Japanese occurred 18 times.
	// Rules chinese occurred 4 times.
	// Rules japanese occurred 2 times.
	// Total Rules games 42 with 4 different Ruless
	// Rank 9p occurred 48 times.
	// Rank 3p occurred 18 times.
	// Rank 7p occurred 16 times.
	// Rank 4p occurred 10 times.
	// Rank 6p occurred 10 times.
	// Rank 5p occurred 8 times.
	// Rank 7k occurred 8 times.
	// Rank 8k occurred 8 times.
	// Rank 8p occurred 8 times.
	// Rank 9 dan occurred 8 times.
	// Rank 1p occurred 2 times.
	// Total Rank games 144 with 11 different Ranks
	// Player Ch'oe Myeong-hun: 2, first: 1006, 4p, last: 1006, 4p
	// Player Ch'oe Weon-yong: 2, first: 1007, 1p, last: 1007, 1p
	// Player Chang Hao: 52, first: 1000, 7p, last: 1025, 6p
	// Player Chen Linxin: 2, first: 1023, 8p, last: 1023, 8p
	// Player Cho Chikun: 4, first: print1, 9 dan, last: print2, 9 dan
	// Player Cho Hun-hyeon: 4, first: 1003, 9p, last: 1018, 9p
	// Player Ding Wei: 2, first: 1021, 5p, last: 1021, 5p
	// Player Dong Yan: 2, first: 1008, 7p, last: 1008, 7p
	// Player Duan Rong: 2, first: 1009, 5p, last: 1009, 5p
	// Player Gu Li: 2, first: 1022, 7p, last: 1022, 7p
	// Player Kobayashi Koichi: 2, first: 1024, 9p, last: 1024, 9p
	// Player Kobayashi Satoru: 2, first: 1010, 9p, last: 1010, 9p
	// Player Kong Jie: 2, first: 1011, 7p, last: 1011, 7p
	// Player Luo Xihe: 2, first: 1012, 9p, last: 1012, 9p
	// Player Lyonweiqi: 16, first: Lyonweiqi-ken1jf-2, 3p, last: Lyonweiqi, 3p
	// Player Qian Yuping: 2, first: 1025, 9p, last: 1025, 9p
	// Player Ryu Shikun: 4, first: 1013, 6p, last: print2, 9 dan
	// Player Shao Weigang: 2, first: 1014, 9p, last: 1014, 9p
	// Player Takemiya Masaki: 2, first: print1, 9 dan, last: print1, 9 dan
	// Player Wang Lei: 2, first: 1015, 4p, last: 1015, 4p
	// Player Wang Qun: 2, first: 1000, 8p, last: 1000, 8p
	// Player Wang Yao: 2, first: 1016, 5p, last: 1016, 5p
	// Player Yamada Kimio: 2, first: 1001, 6p, last: 1001, 6p
	// Player Yi Ch'ang-ho: 2, first: 1002, 9p, last: 1002, 9p
	// Player Yu Bin: 2, first: 1004, 9p, last: 1004, 9p
	// Player Yu Ping: 2, first: 1017, 6p, last: 1017, 6p
	// Player Zhang Xuebin: 2, first: 1019, 4p, last: 1019, 4p
	// Player Zhou Heyang: 4, first: 1005, 6p, last: 1020, 3p
	// Player ken1jf: 16, first: Lyonweiqi-ken1jf-2, 7k, last: Lyonweiqi, 7k
	//  52 : Chang Hao, first:  1000, 7p, last: 1025, 6p
	//  16 : Lyonweiqi, first:  Lyonweiqi-ken1jf-2, 3p, last: Lyonweiqi, 3p
	//  16 : ken1jf, first:  Lyonweiqi-ken1jf-2, 7k, last: Lyonweiqi, 7k
	//  4 : Cho Chikun, first:  print1, 9 dan, last: print2, 9 dan
	//  4 : Ryu Shikun, first:  1013, 6p, last: print2, 9 dan
	//  4 : Zhou Heyang, first:  1005, 6p, last: 1020, 3p
	//  4 : Cho Hun-hyeon, first:  1003, 9p, last: 1018, 9p
	//  2 : Dong Yan, first:  1008, 7p, last: 1008, 7p
	//  2 : Duan Rong, first:  1009, 5p, last: 1009, 5p
	//  2 : Gu Li, first:  1022, 7p, last: 1022, 7p
	//  2 : Kobayashi Koichi, first:  1024, 9p, last: 1024, 9p
	//  2 : Kobayashi Satoru, first:  1010, 9p, last: 1010, 9p
	//  2 : Kong Jie, first:  1011, 7p, last: 1011, 7p
	//  2 : Luo Xihe, first:  1012, 9p, last: 1012, 9p
	//  2 : Chen Linxin, first:  1023, 8p, last: 1023, 8p
	//  2 : Qian Yuping, first:  1025, 9p, last: 1025, 9p
	//  2 : Ch'oe Weon-yong, first:  1007, 1p, last: 1007, 1p
	//  2 : Shao Weigang, first:  1014, 9p, last: 1014, 9p
	//  2 : Takemiya Masaki, first:  print1, 9 dan, last: print1, 9 dan
	//  2 : Wang Lei, first:  1015, 4p, last: 1015, 4p
	//  2 : Wang Qun, first:  1000, 8p, last: 1000, 8p
	//  2 : Wang Yao, first:  1016, 5p, last: 1016, 5p
	//  2 : Yamada Kimio, first:  1001, 6p, last: 1001, 6p
	//  2 : Yi Ch'ang-ho, first:  1002, 9p, last: 1002, 9p
	//  2 : Yu Bin, first:  1004, 9p, last: 1004, 9p
	//  2 : Yu Ping, first:  1017, 6p, last: 1017, 6p
	//  2 : Zhang Xuebin, first:  1019, 4p, last: 1019, 4p
	//  2 : Ch'oe Myeong-hun, first:  1006, 4p, last: 1006, 4p
	//  2 : Ding Wei, first:  1021, 5p, last: 1021, 5p
	//  max Procs set back to 1.
}
