package main

import (
	"flag"
	"fmt"
	"golang.org/x/exp/constraints"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

func main() {
	// filter pattern
	flagPattern := flag.String("p", "", "filter by pattern")
	flagAll := flag.Bool("a", false, "all files including hide files")
	flagNumberRecords := flag.Int("n", 0, "number of records")

	// order flags
	hasOrderByTime := flag.Bool("t", false, "sort by time, oldest first")
	hasOrderBySize := flag.Bool("s", false, "sort by file size, smallest first")
	hasOrderReverse := flag.Bool("r", false, "reverse order while sorting")

	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		path = "."
	}

	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	var fs []file
	for _, f := range files {
		isHidden := isHidden(f.Name(), path)

		if isHidden && !*flagAll {
			continue
		}

		// we check the pattern given in the -p flag
		if *flagPattern != "" {
			isMatch, err := regexp.MatchString("(?i)"+*flagPattern, f.Name())
			if err != nil {
				panic(err)
			}
			if !isMatch {
				continue
			}
		}

		archivo, err := getFile(f, isHidden)
		if err != nil {
			fmt.Println(err)
			return
		}

		fs = append(fs, archivo)
	}

	if !*hasOrderByTime && !*hasOrderBySize {
		orderByName(fs, *hasOrderReverse)
	}

	if *hasOrderBySize && !*hasOrderByTime {
		orderBySize(fs, *hasOrderReverse)
	}

	if *hasOrderByTime && !*hasOrderBySize {
		orderByTime(fs, *hasOrderReverse)
	}

	if *flagNumberRecords == 0 || *flagNumberRecords > len(fs) {
		*flagNumberRecords = len(fs)
	}
	printList(fs, *flagNumberRecords)
}

func mySort[T constraints.Ordered](i, j T, isReverse bool) bool {
	if isReverse {
		return i > j
	}
	return i < j
}

func orderByName(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return mySort(
			strings.ToLower(files[i].name),
			strings.ToLower(files[j].name),
			isReverse,
		)
	})
}

func orderBySize(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return mySort(
			files[i].size,
			files[j].size,
			isReverse,
		)
	})
}

func orderByTime(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return mySort(
			files[i].modificationTime.Unix(),
			files[j].modificationTime.Unix(),
			isReverse,
		)
	})
}

func printList(fs []file, numRegisters int) {
	for _, f := range fs[:numRegisters] {
		style := mapStyleByFileType[f.fileType]

		fmt.Printf("%s %s %s %8d %v %s %s%s\n",
			f.mode, f.userName, f.groupName, f.size, f.modificationTime.Format(time.Stamp),
			style.icon, f.name, style.symbol,
		)
	}
}

// getFile returns a file object for the given file entry.
// It returns an error if it fails to retrieve information about the file.
func getFile(f os.DirEntry, isHidden bool) (file, error) {
	// info returns information about the named file.
	info, err := f.Info()
	if err != nil {
		return file{}, fmt.Errorf("f.Info(): %v", err)
	}

	// create a new file object with the information retrieved from the file entry.
	result := file{
		name:             f.Name(),
		isDir:            f.IsDir(),
		isHidden:         isHidden,
		userName:         "user",
		groupName:        "group",
		size:             info.Size(),
		modificationTime: info.ModTime(),
		mode:             info.Mode().String(),
	}

	// set the file type based on the file properties.
	setFile(&result)
	return result, nil
}

// setFile sets the file type based on the file propertie
func setFile(f *file) {
	switch {
	case isLink(*f):
		f.fileType = fileLink
	case f.isDir:
		f.fileType = fileDirectory
	case isExec(*f):
		f.fileType = fileExecutable
	case isCompress(*f):
		f.fileType = fileCompress
	case isImage(*f):
		f.fileType = fileImage
	default:
		f.fileType = fileRegular
	}
}

// isLink returns true if the file is a symbolic link.
func isLink(f file) bool {
	return strings.HasPrefix(strings.ToUpper(f.mode), "L")
}

// isExec returns true if the file is executable.
// On Windows, it checks if the file name ends with ".exe".
// On other systems, it checks if the file mode contains the "x" permission.
func isExec(f file) bool {
	if runtime.GOOS == Windows {
		return strings.HasSuffix(f.name, exe)
	}
	return strings.Contains(f.mode, "x")
}

// isCompress returns true if the file is compressed.
func isCompress(f file) bool {
	var suffix = []string{deb, zip, gz, tar, rar}

	for _, s := range suffix {
		if strings.HasSuffix(f.name, s) {
			return true
		}
	}
	return false
}

// isImage returns true if the file is an image.
func isImage(f file) bool {
	var suffix = []string{png, jpg, gif}

	for _, s := range suffix {
		if strings.HasSuffix(f.name, s) {
			return true
		}
	}
	return false
}

func isHidden(filename, basePath string) bool {
	return strings.HasPrefix(filename, ".")
}
