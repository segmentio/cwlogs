package ecslogs

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type FuncInfo struct {
	File string
	Func string
	Line int
}

func (info FuncInfo) String() string {
	if info == (FuncInfo{}) {
		return ""
	}
	return info.File + ":" + strconv.Itoa(info.Line) + ":" + info.Func
}

func GuessCaller(skip int, maxDepth int, ignorePackages ...string) (pc uintptr, ok bool) {
	if len(ignorePackages) == 0 {
		pc, _, _, ok = runtime.Caller(skip + 1)
		return
	}

	frames := make([]uintptr, skip+maxDepth)
	frames = frames[:runtime.Callers(2, frames)]

	// Search for the first stack frame that is not in one of the packages that
	// we want to ignore.
	var i int
search:
	for i, pc = range frames {
		var info FuncInfo

		if info, ok = GetFuncInfo(pc); !ok {
			break
		}

		for _, pkg := range ignorePackages {
			if strings.Contains(info.File, pkg) {
				continue search
			}
		}

		// Now that we got out of the packages that we wanted to ignore we need
		// to go up a couple more stack frames if the `skip` value is not zero.
		if i += skip; i < len(frames) {
			pc = frames[i]
		} else {
			if pc, _, _, ok = runtime.Caller(i + 1); !ok {
				break
			}
		}

		ok = true
		return
	}

	pc = 0
	return
}

func GetFuncInfo(pc uintptr) (info FuncInfo, ok bool) {
	var pkg string
	var fp = runtime.FuncForPC(pc)

	if fp == nil {
		return
	}

	pkg, info.Func = parseFuncName(fp.Name())
	info.File, info.Line = fp.FileLine(pc)

	if len(pkg) == 0 {
		pkg = filepath.Base(filepath.Dir(info.File))
	}

	info.File = filepath.Join(pkg, filepath.Base(info.File))
	ok = true
	return
}

func parseFuncName(name string) (pkg string, fn string) {
	if i := strings.LastIndexByte(name, '/'); i <= 0 {
		fn = name
	} else if j := strings.IndexByte(name[i+1:], '.'); j <= 0 {
		fn = name
	} else {
		i += j + 1
		pkg, fn = name[:i], name[i+1:]
	}
	return
}
