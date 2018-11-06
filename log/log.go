package log

import (
	"compress/gzip"
	"fmt"
	"io"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type logFile struct {
	curFile   *os.File
	fileName  string
	sizeFlag  bool
	timeFlag  bool
	compressFlag bool
	filePath  string
	sizeValue int64
	todayDate string
	msgQueue  chan string
	closed    bool
	cnt 	uint32
}

var (
	_defaultFile = NewLogFile(
		LogFileName(func()string{
			return filepath.Base(os.Args[0]) + ".log"
		}()),
		LogFilePath(func()string{
			path, _ :=  filepath.Abs(os.Args[0])
			strings.TrimRight(path, string(filepath.Separator))
			return path + string(filepath.Separator)+"log"
		}()),
		LogFileCompress(false),
		LogFileSize(10, "M"),
		LogFileTime(true),
		)
)


func init(){
	log.SetOutput(_defaultFile)
}

type Option func(file *logFile)

func NewLogFile(options ...Option) *logFile {
	var err error
	logfile := &logFile{
		fileName: "",
		sizeFlag: false,
		timeFlag: false,
		closed:   false,
		msgQueue: make(chan string, 1000),
		cnt:0,
	}

	for _, option := range options {
		option(logfile)
	}

	err = os.MkdirAll(logfile.filePath,06666)
	if err != nil{
		fmt.Printf("\ncreate log file path failed %v\n",err)
	}

	logfile.todayDate = time.Now().Format("2006-01-02")
	//
	if logfile.fileName != "" {
		file, err := os.OpenFile(logfile.filePath+logfile.fileName,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println(err.Error())
		}
		logfile.curFile = file
	} else {
		logfile.curFile = os.Stdout
	}

	go logfile.worker()

	return logfile
}

//设置文件名
func LogFileName(fileName string) Option {
	return func(file *logFile) {
		file.fileName = fileName
	}
}

//设置文件路径
func LogFilePath(path string) Option {
	return func(file *logFile) {
		var slash string = string(os.PathSeparator)
		Path := strings.TrimRight(path,slash)
		dir, _ := filepath.Abs(Path)
		file.filePath = dir + slash
	}
}

//设置文件切割大小
func LogFileSize(size int, unit string) Option {
	return func(file *logFile) {
		file.sizeFlag = true

		switch unit {
		case "K":
			file.sizeValue = int64(size) * 1024
		case "M":
			file.sizeValue = int64(size) * 1024 * 1024
		case "G":
			file.sizeValue = int64(size) * 1024 * 1024 * 1024
		default:
			file.sizeValue = int64(size)
		}
	}
}

//按照天来切割
func LogFileTime(flag bool) Option {
	return func(file *logFile) {
		file.timeFlag = true
	}
}

func LogFileCompress(flag bool)Option{
	return func( file *logFile){
		file.compressFlag = flag
	}
}

//
func (f *logFile) Write(p []byte) (n int, err error) {
	str := (*string)(unsafe.Pointer(&p))
	f.msgQueue <- (*str)
	return len(p), nil
}

//切割文件
func (f *logFile) doRotate() {

	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Println("doRotate %v", rec)
		}
	}()

	if f.curFile == nil {
		fmt.Println("doRotate curFile nil,return")
		return
	}
	prefile := f.curFile
	_, err := prefile.Stat()
	var prefileName string = ""
	if err == nil {
		filePath := f.filePath + f.fileName
		f.closed = true
		err := prefile.Close()
		if err != nil {
			fmt.Println("doRotate close err", err.Error())
		}
		y, m, d := time.Now().Date()
		f.cnt++
		prefileName = filePath + "." + fmt.Sprintf("%.4d%.2d%.2d", y, m, d) + strconv.FormatInt(int64(f.cnt), 10)
		err = os.Rename(filePath, prefileName)
	}

	if f.fileName != "" {
		nextFile, err := os.OpenFile(f.filePath+f.fileName,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			fmt.Println(err.Error())
		}
		f.closed = false
		f.curFile = nextFile
		nowDate := time.Now().Format("2006-01-02")
		f.todayDate = nowDate
	}

	if f.compressFlag == true{
		go f.compressFile(prefileName, prefileName+".gz")
	}
}

func (f *logFile) worker() {
	for f.closed == false {
		msg := <-f.msgQueue
		f.curFile.WriteString(msg)
		if f.sizeFlag == true {
			curInfo, _ := os.Stat(f.filePath + f.fileName)
			if curInfo.Size() >= f.sizeValue {
				f.doRotate()
			}
		}
		nowDate := time.Now().Format("2006-01-02")
		if f.timeFlag == true &&
			nowDate != f.todayDate {
			f.doRotate()
		}
	}
}

func (f *logFile) compressFile(Src string, Dst string) error {
	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Println(rec)
		}
	}()
	newfile, err := os.Create(Dst)
	if err != nil {
		return err
	}
	defer newfile.Close()

	file, err := os.Open(Src)
	if err != nil {
		return err
	}

	zw := gzip.NewWriter(newfile)

	filestat, err := file.Stat()
	if err != nil {
		return nil
	}

	zw.Name = filestat.Name()
	zw.ModTime = filestat.ModTime()
	_, err = io.Copy(zw, file)
	if err != nil {
		return nil
	}

	zw.Flush()
	if err := zw.Close(); err != nil {
		return nil
	}
	file.Close()
	os.Remove(Src)
	return nil
}
