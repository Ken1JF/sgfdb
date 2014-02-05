/*
 *  File:		src/github.com/Ken1JF/sgfdb/sgfdb.go
 *  Project:	abst-hier
 *
 *  Created by Ken Friedenbach on 11/25/09.
 *  Copyright 2009-2014 Ken Friedenbach. All rights reserved.
 */

// Package sgfdb provides functions for reading .sgf file which are
// held in a two level directory hierarchy.
// The Index of the data base is a directory of directories.
// Each directory in the Index can hold multiple .sgf files.
package sgfdb

import (
	"io"
	"io/ioutil"
	"os"
	"time"
	//    "syscall"
	"fmt"
	"github.com/Ken1JF/ah"
	"github.com/Ken1JF/sgf"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

type ActionFunction func(r *DirectoryProcessRequest, fName string, b []byte)

type DBProcessRequest struct {
	// input parameters
	Requester   string // name of function making request
	DBIndexName string // name of Databse root directory
	DBOutName   string // name of Output root directory

	DoMultiCPU bool // run parallel go routines
	ReportCPUs bool // report changing of CPUs
	MaxAtOnce  int  // number of parallel executions, if 0 use NumCPUs

	SkipFiles int
	FileLimit int
	MoveLimit int

	PModeReq         sgf.ParserMode
	FileActionFunc   ActionFunction
	EndDirActionFunc ActionFunction
	EndDBActionFunc  ActionFunction

	NumPerLine int // Number of moves per line for output .sgf files

	// output results
	totalD   int // can be used by Action Functions, i.e. to count directories
	totalF   int // can be used by Action Functions, i.e. to count files
	totalM   int // can be used by Action Functions, i.e. to count moves or tokens
	totalE   int // can be used by Action Functions, i.e. to count errors
	NumCPUs  int
	DBErrors []error // errors accummulated from Index
}

// temporary Hack
// Only referenced by ReadTeachingDirectory (not converted, yet)
// and main() in test_ahgo.go (could use sgf.Default...)
var TheDBReadReq DBProcessRequest

func (dbReq *DBProcessRequest) initDBRequest(caller string,
	dirName string, outDir string,
	doMany bool, repCPUs bool,
	maxParallel int,
	skip int, maxf int, moveMax int,
	pMode sgf.ParserMode,
	fileActF ActionFunction,
	dirActF ActionFunction,
	dbActF ActionFunction,
	numOnLine int) {

	dbReq.Requester = caller
	dbReq.DBIndexName = dirName
	dbReq.DBOutName = outDir

	dbReq.DoMultiCPU = doMany
	dbReq.MaxAtOnce = maxParallel
	dbReq.ReportCPUs = repCPUs

	dbReq.SkipFiles = skip
	dbReq.FileLimit = maxf
	dbReq.MoveLimit = moveMax

	dbReq.PModeReq = pMode
	dbReq.FileActionFunc = fileActF
	dbReq.EndDirActionFunc = dirActF
	dbReq.EndDBActionFunc = dbActF

	dbReq.NumPerLine = numOnLine

	dbReq.NumCPUs = runtime.NumCPU()

	if maxParallel == 0 {
		dbReq.MaxAtOnce = dbReq.NumCPUs
	}
}

func init() {
	TheDBReadReq.initDBRequest("TheDBReadReq Hack", "", "", true, false, 0, 0, 0, 0, sgf.DefaultParserMode, nil, nil, nil, sgf.DefaultNumPerLine)
}

// ReadSGFDatabase is a function that reads each directory
// in the Index directory, applies supplied action functions,
// and calls a function to read files in each directory.

// TODO: replace each pair below ReadSGFDatabase and action functions.

// TODO: replace ReadDatabaseAndBuildPatterns
//      reads directory of directories
//      for each directory calls ReadDirectoryAndBuildPatterns
// TODO: replace ReadDirectoryAndBuildPatterns
//      loops for files in dir (with .sgf)
//      reads file, parses file
//      then ??? (is this a partial trans from C++)

// TODO: replace ReadTeachingDirectory
//      reads a single directory with teaching games (not a data base)

// TODO: modify ExampleReadWriteSGFFile to use ReadSGFDatabase ???
//      test is currently in sgf/sgfio_test.go

// ReadSGFDatabase uses a request server, a result server, and
// channels to allow reading multiple directories in parallel.
// Optionally, the channels can be limited to depth 1, to force
// sequential execution.
func ReadSGFDatabase(rdReq *DBProcessRequest) error {
	// set the number of CPUs to use
	var oldMaxProcs int
	if rdReq.DoMultiCPU {
		oldMaxProcs = runtime.GOMAXPROCS(rdReq.NumCPUs)
		if rdReq.ReportCPUs {
			fmt.Printf(" num CPUs = %d, default max Procs was %d, now set to num CPUs\n\n", rdReq.NumCPUs, oldMaxProcs)
		}
	} else {
		if rdReq.ReportCPUs {
			fmt.Printf(" num CPUs = %d, but multi-processing not enabled.\n\n", rdReq.NumCPUs)
		}
	}

	// restore the number of CPUs to original value
	if oldMaxProcs != 0 {
		if rdReq.ReportCPUs {
			fmt.Printf(" max Procs set back to %d.\n", oldMaxProcs)
		}
		runtime.GOMAXPROCS(oldMaxProcs)
	}
	return nil
}

// variable set at beginning of program
var zero_time time.Time

// print the time to complete an action
func print_time(action string, place string) (tim time.Time) {
	tim = time.Now()
	fmt.Println(action, place, tim)
	return tim
}

// Set to 1 to enable tracing of sgfdb functions:
// TODO: option to set
var TRACE_SGFDB = 0

// TraceRec passes entry time to termination trace routine
type TraceRec struct {
	s string
	t time.Time
}

// trace is the entry trace routine
func trace(s string) TraceRec {
	var t time.Time
	if TRACE_SGFDB == 1 {
		t = print_time("Entering: ", s)
	}
	return TraceRec{s, t}
}

// un is the termination trace routine
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

// DirectoryProcessRequest holds the communication values and channels
// used to execute ProcessDirectory in parallel as a go-routine.
type DirectoryProcessRequest struct {
	// normal parameters
	i     int    // order in Database directory
	dir   string // dir name, if == "", then i == -1 and fileLimit == count of dirs
	dbReq *DBProcessRequest

	// return values
	cntf int // can be used by Action Functions, i.e. to count files
	cntm int // can be used by Action Functions, i.e. to count moves or tokens

	errAct string // action causing error, if any
	err    error  // error if any
	// communication channels
	checkCount int                           // only used by last request
	reply      chan *DirectoryProcessRequest // to send back results
	done       chan bool                     // to signal completion, through first defer
}

func CountMoves(req *DirectoryProcessRequest, fName string, b []byte) {
	req.cntf++ // TODO: decide: empty files are not SGF files. so don't count?
	idx := strings.Index(string(b), ";")
	for idx > 0 {
		req.cntm++
		b = b[idx+1:]
		idx = strings.Index(string(b), ";")
	}
}

func ReportDirCounts(req *DirectoryProcessRequest, fName string, b []byte) {
	req.dbReq.totalF += req.cntf
	req.dbReq.totalM += req.cntm
	if req.errAct != "" {
		fmt.Printf("%3d:%s:%s\n", req.i, req.errAct, req.err)
	} else {
		idx := strings.LastIndex(req.dir, "/")
		fmt.Printf("%3d:%s, files: %d, moves: %d\n", req.i, req.dir[idx+1:], req.cntf, req.cntm)
	}
}

func ReportDBCounts(req *DirectoryProcessRequest, fName string, b []byte) {
	fmt.Printf("Total SGF files = %d, total moves = %d\n",
		req.dbReq.totalF, req.dbReq.totalM)
}

// ProcessDirectory takes a DirectoryProcessRequest
// which has a directory name and a fileLimit (for short testing)
// and returns a count of .sgf files and moves (";" Nodes found).
// If an error occurs, the third value names the action causing the error
// and the fourth return value is the Error.
// Two channels are provided with the request, one for sending back the results,
// and one to signal that the subroutine is complete (at end of trace output).
func ProcessDirectory(req *DirectoryProcessRequest) {
	defer un(trace("ProcessDirectory"), req.done)

	// read the subDirectory to process
	dirFiles, e := ioutil.ReadDir(req.dir)
	if e != nil && e != io.EOF {
		// if there is an error record it, send reply, and return
		req.err = e
		s := "Reading directory: " + req.dir
		req.errAct = s
		req.reply <- req
		return
	}
	// process the files in the subdirectory
	for _, f := range dirFiles {
		// skip entries that are not .sgf files
		if strings.Index(f.Name(), ".sgf") >= 0 {
			// read the SGF file
			b, e := ioutil.ReadFile(req.dir + "/" + f.Name())
			if e != nil && e != io.EOF {
				// if there is an error record it, send reply, and return
				req.err = e
				s := "Reading file: " + req.dir + "/" + f.Name()
				req.errAct = s
				req.reply <- req
				return
			}
			// if the file is not empty
			if len(b) > 0 {
				// call the action funtion
				if req.dbReq.FileActionFunc != nil {
					idx := strings.LastIndex(f.Name(), "/")
					if idx >= 0 {
						req.dbReq.FileActionFunc(req, f.Name()[idx+1:], b)
					} else {
						req.dbReq.FileActionFunc(req, f.Name(), b)
					}
				}
				// check if fileLimit is set
				if req.dbReq.FileLimit > 0 {
					// check if filelimit has been reached
					if req.cntf >= req.dbReq.FileLimit {
						break
					}
				}
			}
		}
	}
	if req.dbReq.EndDirActionFunc != nil {
		req.dbReq.EndDirActionFunc(req, "", nil)
	}
	req.reply <- req
	return
}

// requestServer runs as an independent go-routine.
// It receives DirectoryProcessRequest records from a reqChan,
// and dispatches them to ProcessDirectory,
// except for the final request, with dir == "", which it sends directly to the replyChan.
func requestServer(reqChan chan *DirectoryProcessRequest, replyChan chan *DirectoryProcessRequest, doneChan chan bool) {
	defer un(trace("requestServer"), nil)
	nRequests := 0
	for {
		<-doneChan         // wait for a procss to be available
		req := <-reqChan   // wait for request to arrive
		if req.dir == "" { // last request has empty directory name.
			if nRequests != req.checkCount {
				fmt.Printf("Error, server request count %d, does not match generator count %d\n", nRequests, req.checkCount)
				if req.dbReq.FileLimit < nRequests {
					nRequests = req.dbReq.FileLimit
				}
			}
			replyChan <- req // send the last request straight to the replyServer, to know when to stop
			break
		}
		nRequests++
		go ProcessDirectory(req)
	}
	return
}

// resultServer runs as an independent go-routine.
// It receives DirectoryProcessRequest records from ProcessDirectory,
// and tallies them,
// except for a special final request, with dir == "", which is sent directly from requestServer.
func resultServer(replyChan chan *DirectoryProcessRequest, doneChan chan bool, finishChan chan bool) {
	defer un(trace("resultServer"), finishChan)
	var req *DirectoryProcessRequest
	expected := -1 // the number of requests counted by generator and request server
	counted := 0
	for {
		req = <-replyChan
		if req.i == -1 && req.dir == "" { // special request passing end info
			expected = req.checkCount
		} else { // normal result
			counted++
		}
		if 0 <= expected && expected <= counted {
			break
		}
	}
	if req.dbReq.EndDBActionFunc != nil {
		req.dbReq.EndDBActionFunc(req, "", nil)
	}
}

// startServers creates the channels needed for communication,
// and launches the request and result Servers.
// It also "primes" the doneChan with enough completion notices to allow the indicated
// amount of parallel execution.
func startServers(dbReq *DBProcessRequest) (reqChan chan *DirectoryProcessRequest, replyChan chan *DirectoryProcessRequest, doneChan chan bool, finishChan chan bool) {
	defer un(trace("startServers"), nil)
	if dbReq.MaxAtOnce > dbReq.NumCPUs {
		dbReq.MaxAtOnce = dbReq.NumCPUs
	}
	if dbReq.DoMultiCPU == false || dbReq.MaxAtOnce == 0 {
		dbReq.MaxAtOnce = 1
	}
	reqChan = make(chan *DirectoryProcessRequest, dbReq.MaxAtOnce)
	replyChan = make(chan *DirectoryProcessRequest, dbReq.MaxAtOnce)
	doneChan = make(chan bool, dbReq.MaxAtOnce)
	finishChan = make(chan bool)

	for i := 1; i <= dbReq.MaxAtOnce; i++ {
		doneChan <- true // signal completions to get parallel execution started
	}

	go resultServer(replyChan, doneChan, finishChan)
	go requestServer(reqChan, replyChan, doneChan)

	return reqChan, replyChan, doneChan, finishChan
}

// ProcessDatabase reads the Database directory, starts the servers,
// builds the requests, and sends them to the requestServer.
// After sending a special final request, it waits for the finishChan to signal completion.
func ProcessDatabase(dbrq *DBProcessRequest) int {
	// Read the sgfdb directories:
	dirs, err := ioutil.ReadDir(dbrq.DBIndexName)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading sgfdb directory: %s, %s\n", dbrq.DBIndexName, err)
		return 2
	}
	reqChan, replyChan, doneChan, finishChan := startServers(dbrq)
	nRequests := 0
	//	errCount := 0;
	// Loop:
	for _, d := range dirs {
		if len(d.Name()) > 0 && d.Name()[0] != '.' {
			fileInfo, err := os.Stat(dbrq.DBIndexName + d.Name())
			if err == nil {
				if fileInfo.IsDir() {
					req := DirectoryProcessRequest{i: nRequests, dir: dbrq.DBIndexName + d.Name(), dbReq: dbrq,
						cntf: 0, cntm: 0,
						errAct: "", err: nil, reply: replyChan, done: doneChan}
					reqChan <- &req
					nRequests++
				}
			}
		}
	}
	// send end packet
	req := DirectoryProcessRequest{i: -1, dir: "", dbReq: dbrq, checkCount: nRequests}
	// fileLimit: nRequests
	reqChan <- &req
	// wait for finished signal from resultServer
	<-finishChan
	return 0
}

// CountFilesAndMoves calls ProcessDatabase with CountMoves as the action function.
func CountFilesAndMoves(db_dir string, fileLimit int, runParalParallel bool, pmode sgf.ParserMode) int {
	defer un(trace("CountFilesAndMoves"), nil)
	var dbReq DBProcessRequest

	dbReq.initDBRequest("CountFilesAndMoves", db_dir, "", runParalParallel, false, 1, 0, fileLimit, 0, pmode, CountMoves, ReportDirCounts, ReportDBCounts, sgf.DefaultNumPerLine)
	ret := ProcessDatabase(&dbReq)
	return ret
}

func WriteSGFDirectory(r *DirectoryProcessRequest, fName string, b []byte) {
	d := r.dir
	idx := strings.LastIndex(d, "/")
	if idx >= 0 {
		d = d[idx+1:]
	}
	fmt.Printf("%3d:%s, files: %d, tokens: %d", r.i, d, r.cntf, r.cntm)
	if r.err != nil {
		fmt.Printf("error: %s%s\n", r.errAct, r.err)
	} else {
		fmt.Printf("\n")
	}

	r.dbReq.totalD += 1
	r.dbReq.totalF += r.cntf
	r.dbReq.totalM += r.cntm
	if r.err != nil {
		r.dbReq.totalE += 1
	}
}

func WriteSGFDatabase(r *DirectoryProcessRequest, fName string, b []byte) {
	fmt.Printf("Total SGF files = %d, tokens = %d", r.dbReq.totalF, r.dbReq.totalM)
	if r.dbReq.totalE > 0 {
		fmt.Printf("errors: %d\n", r.dbReq.totalE)
	} else {
		fmt.Printf("\n")
	}
}

//
func WriteSGFFile(r *DirectoryProcessRequest, fName string, b []byte) {

	fullFileName := r.dir + "/" + fName
	prsr, errL := sgf.ParseFile(fullFileName, b, r.dbReq.PModeReq, r.dbReq.MoveLimit)
	r.cntf += 1
	if len(errL) != 0 {
		fmt.Printf("%s Error(s) during parsing: %s\n", r.dbReq.Requester, fullFileName)
		ah.PrintError(os.Stdout, errL)
		return // cntF, cntT, cntE, errL // stop on first error?
	}
	inDir := r.dir
	idx := strings.LastIndex(inDir, "/")
	if idx >= 0 {
		inDir = inDir[idx+1:]
	}
	outDir := r.dbReq.DBOutName + inDir
	outFileName := outDir + "/" + fName
	// Check the output directory. If missing, create it.
	_, errS := os.Stat(outDir)
	if errS != nil {
		err2 := os.MkdirAll(outDir, os.ModeDir|os.ModePerm)
		if err2 != nil {
			fmt.Println(r.dbReq.Requester, "Error:", err2, "trying to create test output directory:", outDir)
			fmt.Println("Original Error:", errS, "trying os.Stat")
			return // cntF, cntT, cntE, err2 // stop on first error?
		}
	}
	err := prsr.GameTree.WriteFile(outFileName, r.dbReq.NumPerLine)
	if err != nil {
		fmt.Printf("%s Error writing: %s, %s\n", r.dbReq.Requester, outFileName, err)
		return // cntF, cntT, cntE, err
	}
}

// ReadDirectoryAndBuildPatterns
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
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.TraceParser, moveLimit)
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.ParserPlay+sgf.ParserDbStat, moveLimit)
					//			prsr,errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.ParserGoGoD, moveLimit)
					/* prsr */ _, errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.ParserGoGoD+sgf.ParserPlay, moveLimit)
					if len(errL) != 0 {
						fmt.Printf("Error(s) during parsing: %s\n", fileName)
						ah.PrintError(os.Stdout, errL)
						return patternTree, err // stop on first error?
					}
					/* TODO: replace this "write output" logic with pattern add logic
					   if outDir != "" {
					       outFileName := outDir + "/" + f.Name()
					       err = prsr.GameTree.WriteFile(outFileName, dbReq.NumPerLine)
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

// ReadAndWriteDatabase builds a DBProcessRequest
// and passes it to ProcessDataBase.
func ReadAndWriteDatabase(db_dir string, testout_dir string, fileLimit int, moveLimit int, skipFiles int, pMode sgf.ParserMode) int {

	defer un(trace("ReadAndWriteDatabase"), nil)

	var dbReq DBProcessRequest

	fmt.Printf("Reading and writing database, db_dir = %v, testout_dir = %v\n",
		db_dir, testout_dir)

	dbReq.initDBRequest("ReadAndWriteDatabase", db_dir, testout_dir, false, false, 1, skipFiles, fileLimit, moveLimit, pMode|sgf.ParserGoGoD|sgf.ParserPlay, WriteSGFFile, WriteSGFDirectory, WriteSGFDatabase, sgf.DefaultNumPerLine)
	ret := ProcessDatabase(&dbReq)
	return ret
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
				prsr, errL := sgf.ParseFile(fileName, b, sgf.ParseComments+sgf.ParserGoGoD+sgf.ParserPlay, moveLimit)
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
			// TODO: is NumPerLine the right number?
			haWholeBoards[i].WriteFile(teachPatsDir+str, TheDBReadReq.NumPerLine)
			fmt.Printf("Patterns written to: %s%s\n", teachPatsDir, str)
		}
	}
	fmt.Printf("Total Handicap games: %d with %d different handicaps\n", sum, count)
	return 0
}

// Some functions to print the size and alignment of types:
func printSizeAlign(s string, sz uintptr, al uintptr) {
	fmt.Println("Type", s, "size", sz, "alignment", al)
}

func PrintSgfDbTypeSizes() {
	// sgfdb.go
	var tr TraceRec
	var cdr DirectoryProcessRequest
	printSizeAlign("TraceRec", unsafe.Sizeof(tr), unsafe.Alignof(tr))
	printSizeAlign("DirectoryProcessRequest", unsafe.Sizeof(cdr), unsafe.Alignof(cdr))
}
