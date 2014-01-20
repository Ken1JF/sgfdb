/*
 *  File:		src/gitHub.com/Ken1JF/ahgo/sgfdb/sgfdb.go
 *  Project:	abst-hier
 *
 *  Created by Ken Friedenbach on 11/25/09.
 *  Copyright 2009-2014 Ken Friedenbach. All rights reserved.
 */

package sgfdb

import (
	"io"
	"io/ioutil"
	"os"
	"time"
	//    "syscall"
	"fmt"
	"gitHub.com/Ken1JF/ahgo/ah"
	"gitHub.com/Ken1JF/ahgo/sgf"
	"strconv"
	"strings"
	"unsafe"
)

// number of parallel executions
//
const MAX_AT_ONCE = 4

// Number of moves per line in SGFDB .sgf files
const SGFDB_NUM_PER_LINE = 12

// variable set at beginning of program
//
var zero_time time.Time

// print the time to complete an action
//
func print_time(action string, place string) (tim time.Time) {
	tim = time.Now()
	fmt.Println(action, place, tim)
	return tim
}

// Set to 1 to enable tracing of sgfdb functions:
// TODO: option to set
//
var TRACE_SGFDB = 0

// TraceRec passes entry time to termination trace routine
//
type TraceRec struct {
	s string
	t time.Time
}

// trace is the entry trace routine
//
func trace(s string) TraceRec {
	var t time.Time
	if TRACE_SGFDB == 1 {
		t = print_time("Entering: ", s)
	}
	return TraceRec{s, t}
}

// un is the termination trace routine
//
func un(r TraceRec, done chan bool) {
	if TRACE_SGFDB == 1 {
		t1 := print_time("Leaving: ", r.s)
		dur := t1.Sub(r.t)
		fmt.Println("Duration:", dur)
	}
	if done != nil { // test for nil for non-go routines
		done <- true // signal completion, allow another to run
	}
}

// CountDirRequest holds the communication values and channels used to execute
// CountDirectory in parallel as a go-routine.
//
type CountDirRequest struct {
	// normal parameters
	i         int    // order in Database directory
	dir       string // dir name, if == "", then i == -1 and fileLimit == count of dirs
	fileLimit int    // fileLimit on files to read, 0 => no fileLimit
	// return values
	cntf int     // count of files read
	cntm int     // count of Nodes/moves (";")
	act  *string // action causing error, if any
	err  error   // error if any
	// communication channels
	reply chan *CountDirRequest // to send back results
	done  chan bool             // to signal completion, through first defer
}

// CountDirectory takes a directory name and a fileLimit (for short testing)
// and returns a count of .sgf files and moves (";" Nodes found).
// If an error occurs, the third value names the action causing the error
// and the fourth return value is the Error.
// Two channels are provided with the request, one for sending back the results,
// and one to signal that the subroutine is complete (at end of trace output).
//
func CountDirectory(req *CountDirRequest) {
	defer un(trace("CountDirectory"), req.done) // channel used for flow control and synchronization
	dirFiles, e := ioutil.ReadDir(req.dir)
	if e != nil && e != io.EOF {
		req.err = e
		s := "Reading directory: " + req.dir
		req.act = &s
		req.reply <- req
		return
	}
	for _, f := range dirFiles {
		if strings.Index(f.Name(), ".sgf") >= 0 {
			b, e := ioutil.ReadFile(req.dir + "/" + f.Name())
			if e != nil && e != io.EOF {
				req.err = e
				s := "Reading file: " + req.dir + "/" + f.Name()
				req.act = &s
				req.reply <- req
				return
			}
			if len(b) > 0 {
				req.cntf++ // TODO: decide: empty files are not SGF files. so don't count?
				idx := strings.Index(string(b), ";")
				for idx > 0 {
					req.cntm++
					b = b[idx+1:]
					idx = strings.Index(string(b), ";")
				}
				if req.fileLimit > 0 {
					if req.cntf >= req.fileLimit {
						break
					}
				}
			}
		}
	}
	req.reply <- req
	return
}

// requestServer runs as an independent go-routine.
// It receives CountDirRequest records from a reqChan, and dispatches them to CountDirectory,
// except for the final request, with dir == "", which it sends directly to the replyChan.
// TODO: make CountDirectory a parameter, so that ReadAndWriteDirectory can be run in parallel, too.
//
func requestServer(reqChan chan *CountDirRequest, replyChan chan *CountDirRequest, doneChan chan bool) {
	defer un(trace("requestServer"), nil)
	nRequests := 0
	for {
		<-doneChan         // wait for a procss to be available
		req := <-reqChan   // wait for request to arrive
		if req.dir == "" { // last request has empty directory name.
			if nRequests != req.fileLimit {
				fmt.Printf("Error, server request count %d, does not match generator count %d\n", nRequests, req.fileLimit)
				if req.fileLimit < nRequests {
					nRequests = req.fileLimit
				}
			}
			replyChan <- req // send the last request straight to the replyServer, to know when to stop
			break
		}
		nRequests++
		go CountDirectory(req)
	}
	return
}

// resultServer runs as an independent go-routine.
// It receives CountDirRequest records from CountDirectory, and tallies them,
// except for a special final request, with dir == "", which is sent directly from requestServer.
//
func resultServer(replyChan chan *CountDirRequest, doneChan chan bool, finishChan chan bool) {
	defer un(trace("resultServer"), finishChan)
	var totalF, totalM int
	expected := -1 // the number of requests counted by generator and request server
	counted := 0
	for {
		req := <-replyChan
		if req.i == -1 && req.dir == "" { // special request passing end info
			expected = req.fileLimit
		} else { // normal result
			totalF += req.cntf
			totalM += req.cntm
			if req.act != nil {
				fmt.Printf("%3d:%s:%s\n", req.i, *req.act, req.err)
			} else {
				idx := strings.LastIndex(req.dir, "/")
				fmt.Printf("%3d:%s, files: %d, moves: %d\n", req.i, req.dir[idx+1:], req.cntf, req.cntm)
			}
			counted++
		}
		if 0 <= expected && expected <= counted {
			break
		}
	}
	fmt.Printf("Total SGF files = %d, total moves = %d\n", totalF, totalM)
}

// startServers creates the channels needed for communication,
// and launches the request and result Servers.
// It also "primes" the doneChan with enough completion notices to allow the indicated
// amount of parallel execution.
//
func startServers() (reqChan chan *CountDirRequest, replyChan chan *CountDirRequest, doneChan chan bool, finishChan chan bool) {
	defer un(trace("startServers"), nil)
	reqChan = make(chan *CountDirRequest, MAX_AT_ONCE)
	replyChan = make(chan *CountDirRequest, MAX_AT_ONCE)
	doneChan = make(chan bool, MAX_AT_ONCE)
	finishChan = make(chan bool)

	for i := 1; i <= MAX_AT_ONCE; i++ {
		doneChan <- true // signal completions to get parallel execution started
	}

	go resultServer(replyChan, doneChan, finishChan)
	go requestServer(reqChan, replyChan, doneChan)

	return reqChan, replyChan, doneChan, finishChan
}

// CountFilesAndMoves reads the Database directory, starts the servers,
// builds the requests, and sends them to the requestServer.
// After sending a special final request, it waits for the finishChan to signal completion.
//
func CountFilesAndMoves(db_dir string, fileLimit int) int {
	defer un(trace("CountFilesAndMoves"), nil)
	// Read the sgfdb directories:
	dirs, err := ioutil.ReadDir(db_dir)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading sgfdb directory: %s, %s\n", db_dir, err)
		return 2
	}
	reqChan, replyChan, doneChan, finishChan := startServers()
	nRequests := 0
	//	errCount := 0;
	// Loop:
	for i, d := range dirs {
		if len(d.Name()) > 0 && d.Name()[0] != '.' {
			req := CountDirRequest{i: i, dir: db_dir + d.Name(), fileLimit: fileLimit, cntf: 0, cntm: 0, act: nil, err: nil, reply: replyChan, done: doneChan}
			reqChan <- &req
			nRequests++
		}
	}
	// send end packet
	req := CountDirRequest{i: -1, dir: "", fileLimit: nRequests}
	reqChan <- &req
	// wait for finished signal from resultServer
	<-finishChan
	return 0
}

// ReadAndWriteDirectory reads the directory passed as the first parameter.
// The second parameter limits the number of files processed (0 means no fileLimit)
// After reading the directory, ReadAndWriteDirectory trys to read and write
// each ".sgf" file found. (Directory entries beginning in "." are skipped.)
// The result is:
//		count of the files read
//		count of the tokens read
//		count of the SGF errors encounterd
//		any Error encounterd during i/o operations.
//
// TODO: Currently the function returns on the first error encountered.
// TODO: Implement flags to allow processing to continue after an error.
//
func ReadAndWriteDirectory(dir string, outDir string, fileLimit int, moveLimit int, skipFiles int) (cntF int, cntT int, cntE int, err error) {
	defer un(trace("ReadAndWriteDirectory"), nil)
	dirFiles, err := ioutil.ReadDir(dir)
	if err != nil && err != io.EOF {
		return cntF, cntT, cntE, err
	}

	for _, f := range dirFiles {
		if strings.Index(f.Name(), ".sgf") >= 0 {

			fileName := dir + "/" + f.Name()
			b, err := ioutil.ReadFile(fileName)
			if err != nil && err != io.EOF {
				fmt.Printf("Error reading File: %s, %s\n", fileName, err)
				return cntF, cntT, cntE, err
			}
			// Use first call to turn on tracing, second to play while reading, third for GoGoD checking:
			// and combinations.
			//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.Trace, moveLimit)
			//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.Play, moveLimit)
			//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.GoGoD, moveLimit)
			prsr, errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.GoGoD+sgf.Play, moveLimit)
			cntF++
			//			cntT += nTok;
			//			cntE += nErr;
			if len(errL) != 0 {
				fmt.Printf("Error(s) during parsing: %s\n", fileName)
				ah.PrintError(os.Stdout, errL)
				return cntF, cntT, cntE, errL // stop on first error?
			}
			if outDir != "" {
				outFileName := outDir + "/" + f.Name()
				err = prsr.GameTree.WriteFile(outFileName, SGFDB_NUM_PER_LINE)
				if err != nil {
					fmt.Printf("Error writing: %s, %s\n", outFileName, err)
					return cntF, cntT, cntE, err
				}
			}
			if fileLimit > 0 {
				if cntF >= fileLimit {
					break
				}
			}
		}
	}
	return cntF, cntT, cntE, err
}

// ReadDirectoryAndBuildPatterns
//
func ReadDirectoryAndBuildPatterns(dir_Name string, subDir_Name string, Pattern_dir string, patternTree *sgf.GameTree, pattern_typ ah.PatternType, fileLimit int, moveLimit int, skipFiles int) (*sgf.GameTree, error) {
	defer un(trace("ReadDirectoryAndBuildPatterns"), nil)
	var err error
	fmt.Printf("Reading directory %s\n", subDir_Name)
	fils, err := ioutil.ReadDir(dir_Name + subDir_Name)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading Database directory: %s, %s\n", dir_Name+subDir_Name, err)
		return patternTree, err
	}
	filesRead := 0
	for i, fil := range fils {
		if strings.Index(fil.Name(), ".sgf") >= 0 {
			if skipFiles > 0 {
				fmt.Printf("Skipping file index %d: %s\n", i, fil.Name())
				skipFiles -= 1
			} else {
				if fileLimit == 0 || filesRead < fileLimit {
					filesRead += 1
					fmt.Printf("Reading file %d, index %d: %s\n", filesRead, i, fil.Name())
					fileName := dir_Name + subDir_Name + "/" + fil.Name()
					b, err := ioutil.ReadFile(fileName)
					if err != nil && err != io.EOF {
						fmt.Printf("Error reading File: %s, %s\n", fileName, err)
						return patternTree, err
					}
					// Use first call to turn on tracing, second to play while reading, third for GoGoD checking:
					// and combinations.
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.Trace, moveLimit)
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.Play, moveLimit)
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.GoGoD, moveLimit)
					/* prsr */ _, errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.GoGoD+sgf.Play, moveLimit)
					if len(errL) != 0 {
						fmt.Printf("Error(s) during parsing: %s\n", fileName)
						ah.PrintError(os.Stdout, errL)
						return patternTree, err // stop on first error?
					}
					/* TODO: replace this "write output" logic with pattern add logic
					   if outDir != "" {
					       outFileName := outDir + "/" + f.Name()
					       err = prsr.GameTree.WriteFile(outFileName, SGFDB_NUM_PER_LINE)
					       if err != nil {
					           fmt.Printf("Error writing: %s, %s\n", outFileName, err)
					           return patternTree, err
					       }
					   }
					*/

				} else {
					fmt.Printf("file limit reached: %d\n", filesRead)
					break
				}
			}
		}
	}

	return patternTree, err
}

// ReadDatabaseAndBuildPatterns
//
func ReadDatabaseAndBuildPatterns(db_dir string, pattern_dir string, pattern_typ ah.PatternType, fileLimit int, moveLimit int, skipFiles int) (status int) {
	defer un(trace("ReadDatabaseAndBuildPatterns"), nil)
	dirs, err := ioutil.ReadDir(db_dir)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading sgfdb directory: %s, %s\n", db_dir, err)
		return 2
	}
	fmt.Printf("Reading database directory: %s for %s\n", db_dir, pattern_dir)
	var patternTree *sgf.GameTree
	// TODO: logic to see if there is one on disk, and read it?
	for i, dir := range dirs {
		if dir.IsDir() && dir.Name()[0] != '.' {
			fmt.Printf("Reading directory %d. %s \n", i, dir.Name())
			patternTree, err = ReadDirectoryAndBuildPatterns(db_dir, dir.Name(), pattern_dir, patternTree, pattern_typ, fileLimit, moveLimit, skipFiles)
			if err != nil {
				fmt.Printf("Error reading directory: %s, %s\n", dir.Name(), err)
			}
		} else {
			fmt.Printf("Not a subdirectory: %d. %s \n", i, dir.Name())
		}
	}
	return status
}

// ReadAndWriteDatabase reads the Database directory, and for each directory,
// calls ReadAndWriteDirectory.
// TODO: rewrite to use the request and result servers.
//
func ReadAndWriteDatabase(db_dir string, testout_dir string, fileLimit int, moveLimit int, skipFiles int) (status int) {
	defer un(trace("ReadAndWriteDatabase"), nil)
	// Read the sgfdb directories:
	var total_files, total_tokens, total_errors int
	fmt.Printf("Reading and writing database, db_dir = %v, testout_dir = %v\n",
		db_dir, testout_dir)
	dirs, err := ioutil.ReadDir(db_dir)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading sgfdb directory: %s, %s\n", db_dir, err)
		return 2
	}
	// turn on tracing:
	//	ah.SetAHTrace(true)
	for i, dir := range dirs {
		if dir.Name()[0] != '.' {
			nf, nt, ne, err /*, Cnts */ := ReadAndWriteDirectory(db_dir+dir.Name(), testout_dir+dir.Name(), fileLimit, moveLimit, skipFiles)
			//			DirCnts[i] = Cnts
			if err != nil {
				fmt.Printf("%3d:%s, not a directory %s\n", i, dir.Name(), err)
			} else {
				idx := strings.LastIndex(dir.Name(), "/")
				fmt.Printf("%3d:%s, files: %d, tokens: %d", i, dir.Name()[idx+1:], nf, nt)
				if ne > 0 {
					fmt.Printf("errors: %d\n", ne)
				} else {
					fmt.Printf("\n")
				}
				total_files += nf
				total_tokens += nt
				total_errors += ne
			}
		}
	}
	fmt.Printf("Total SGF files = %d, tokens = %d", total_files, total_tokens)
	if total_errors > 0 {
		fmt.Printf("errors: %d\n", total_errors)
	} else {
		fmt.Printf("\n")
	}
	return 0
}

func ReadTeachingDirectory(teachDir string, teachPatsDir string, fileLimit int, moveLimit int, patternLimit int, skipFiles int) (status int) {
	defer un(trace("ReadTeachingDirectory"), nil)
	var haCounts [10]int
	var haWholeBoards [10]*sgf.GameTree

	fils, err := ioutil.ReadDir(teachDir)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading Teaching directory: %s, %s\n", teachDir, err)
		return 2
	}
	nFils := 0
	for i, fil := range fils {
		if strings.Index(fil.Name(), ".sgf") >= 0 {
			if skipFiles > 0 {
				skipFiles -= 1
			} else {
				fileName := teachDir + fil.Name()
				b, err := ioutil.ReadFile(fileName)
				if err != nil && err != io.EOF {
					fmt.Printf("Error reading teaching File %d: %s, %s\n", i, fileName, err)
					return 3
				}
				prsr, errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.GoGoD+sgf.Play, moveLimit)
				if len(errL) != 0 {
					fmt.Printf("Error %s during parsing: %s\n", errL.Error(), fileName)
					return 4
				}
				// add logic go processing teaching file
				nFils += 1
				fmt.Printf("%d: Read file no. %d %s ", nFils, i, fileName)
				ha := prsr.GetHA()
				nCol, nRow := prsr.GetSize()
				haCounts[ha] += 1
				fmt.Printf(" %d handicap on %d x %d board.\n", ha, nCol, nRow)

				errL, trans, newPatt := prsr.AddTeachingPattern(nCol, nRow, ha, haWholeBoards[ha], ah.WHOLE_BOARD_PATTERN, moveLimit, patternLimit, skipFiles)
				if len(errL) != 0 {
					fmt.Printf("Error adding Teaching Pattern %s\n", errL.Error())
					return 5
				} else { // if no err, update
					haWholeBoards[ha] = newPatt
				}
				fmt.Printf(" %s transformation to canonical first move.\n", ah.TransName[trans])
				if (fileLimit > 0) && (nFils >= fileLimit) {
					break
				}
			}
		}
	}
	sum := 0
	count := 0
	for i, n := range haCounts {
		if n > 0 {
			sum += n
			count += 1
			fmt.Printf("Handicap %d occurred %d times.\n", i, n)
			str := "HA_" + strconv.Itoa(i) + ".sgf"
			// TODO: is SGFDB_NUM_PER_LINE the right number?
			haWholeBoards[i].WriteFile(teachPatsDir+str, SGFDB_NUM_PER_LINE)
			fmt.Printf("Patterns written to: %s%s\n", teachPatsDir, str)
		}
	}
	fmt.Printf("Total Handicap games: %d with %d different handicaps\n", sum, count)
	return 0
}

// Some functions to print the size and alignment of types:
//
//func printSizeAlign(s string, sz int, al int) {
func printSizeAlign(s string, sz uintptr, al uintptr) {
	fmt.Println("Type", s, "size", sz, "alignment", al)
}

func PrintSgfDbTypeSizes() {
	// sgfdb.go
	var tr TraceRec
	var cdr CountDirRequest
	printSizeAlign("TraceRec", unsafe.Sizeof(tr), unsafe.Alignof(tr))
	printSizeAlign("CountDirRequest", unsafe.Sizeof(cdr), unsafe.Alignof(cdr))
}
