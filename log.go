package hms

import (
	"container/ring"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// These flags define which text to prefix to each log entry generated by the Logger.
// Bits are or'ed together to control what's printed.
// There is no control over the order they appear (the order listed
// here) or the format they present (as described in the comments).
// The prefix is followed by a colon only when Llongfile or Lshortfile
// is specified.
// For example, flags Ldate | Ltime (or LstdFlags) produce,
//	2009/01/23 01:23:23 message
// while flags Ldate | Ltime | Lmicroseconds | Llongfile produce,
//	2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

// LogItem represents structured log fields for each log entry.
// It's used to transmit the log items by network.
type LogItem struct {
	Time    int64  `json:"time"`
	Message string `json:"msg"`
	Level   string `json:"level"`
	Line    int    `json:"line"`
	File    string `json:"file"`
}

// A Logger represents an active logging object that generates lines of
// output to an io.Writer. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
type Logger struct {
	mux  sync.Mutex // ensures atomic writes; protects the following fields
	out  io.Writer  // destination for output
	ring *ring.Ring // for accumulating last log items
	lim  int        // maximum log size
	size int        // current log size
	flag int        // properties
}

// New creates a new Logger. The out variable sets the
// destination to which log data will be written.
// The flag argument defines the logging properties.
func NewLogger(out io.Writer, flag int, lim int) *Logger {
	return &Logger{
		out:  out,
		flag: flag,
		lim:  lim,
	}
}

// SetOutput sets the output destination for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	defer l.mux.Unlock()
	l.mux.Lock()
	l.out = w
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	var bp = len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		var q = i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// formatHeader writes log header to buf in following order:
//   * date and/or time (if corresponding flags are provided),
//   * file and line number (if corresponding flags are provided).
func formatHeader(buf *[]byte, t time.Time, flag int, file string, line int) {
	if flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if flag&LUTC != 0 {
			t = t.UTC()
		}
		if flag&Ldate != 0 {
			var year, month, day = t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if flag&(Ltime|Lmicroseconds) != 0 {
			var hour, min, sec = t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if flag&(Lshortfile|Llongfile) != 0 {
		if flag&Lshortfile != 0 {
			var short = file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
}

// Output writes the output for a logging event. The string s contains
// the text to print after the prefix specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Calldepth is used to recover the PC and is
// provided for generality, although at the moment on all pre-defined
// paths it will be 2.
func (l *Logger) Output(calldepth int, lev string, s string) error {
	var now = time.Now() // get this early.
	var flag = l.Flags()
	var file string
	var line int
	if flag&(Lshortfile|Llongfile) != 0 {
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
	}

	var buf []byte
	formatHeader(&buf, now, flag, file, line)
	buf = append(buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}
	var li = LogItem{
		Time:    UnixJS(now),
		Message: s,
		Level:   lev,
		Line:    line,
		File:    file,
	}

	defer l.mux.Unlock()
	l.mux.Lock()
	if l.ring != nil {
		if l.size < l.lim {
			var r = ring.New(1)
			r.Value = li
			l.ring.Link(r)
			l.ring = r
			l.size++
		} else {
			l.ring = l.ring.Next()
			l.ring.Value = li
		}
	} else {
		var r = ring.New(1)
		r.Value = li
		l.ring = r
		l.size++
	}
	var _, err = l.out.Write(buf)
	return err
}

// Logf calls l.Output to print to the logger with specified level.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Logf(level string, format string, v ...interface{}) {
	l.Output(2, level, fmt.Sprintf(format, v...))
}

// Log calls l.Output to print to the logger with specified level.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Log(level string, v ...interface{}) { l.Output(2, level, fmt.Sprint(v...)) }

// Logln calls l.Output to print to the logger with specified level.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Logln(level string, v ...interface{}) { l.Output(2, level, fmt.Sprintln(v...)) }

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Output(2, "info", fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) { l.Output(2, "info", fmt.Sprint(v...)) }

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(v ...interface{}) { l.Output(2, "info", fmt.Sprintln(v...)) }

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	l.Output(2, "fatal", fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(2, "fatal", fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	l.Output(2, "fatal", fmt.Sprintln(v...))
	os.Exit(1)
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(v ...interface{}) {
	var s = fmt.Sprint(v...)
	l.Output(2, "panic", s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	var s = fmt.Sprintf(format, v...)
	l.Output(2, "panic", s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	var s = fmt.Sprintln(v...)
	l.Output(2, "panic", s)
	panic(s)
}

// Returns last element of ring of log items.
// Ring must be viewed only in backward order.
func (l *Logger) Ring() *ring.Ring {
	defer l.mux.Unlock()
	l.mux.Lock()
	return l.ring
}

// Returns current size of ring of log items.
// Ring must be viewed only in backward order.
func (l *Logger) Size() int {
	defer l.mux.Unlock()
	l.mux.Lock()
	return l.size
}

// Flags returns the output flags for the logger.
func (l *Logger) Flags() int {
	defer l.mux.Unlock()
	l.mux.Lock()
	return l.flag
}

// SetFlags sets the output flags for the logger.
func (l *Logger) SetFlags(flag int) {
	defer l.mux.Unlock()
	l.mux.Lock()
	l.flag = flag
}

// Writer returns the output destination for the logger.
func (l *Logger) Writer() io.Writer {
	defer l.mux.Unlock()
	l.mux.Lock()
	return l.out
}

// The End.
