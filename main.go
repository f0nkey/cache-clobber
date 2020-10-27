package main

import (
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	baseDir := flag.String("dir", "./", "specifies the directory to scan recursively in for html files")

	flag.Parse()

	changes := appendHashes(*baseDir)
	changes.printChangesErrors()
}

type changes struct {
	edits  map[string][]edit // [htmlFile]edits
	errors map[string][]editError
}

type edit struct {
	fileNameFrom string
	fileNameTo   string
	htmlFile     string
}

type editError struct {
	err      error
	htmlFile string
}

func (c *changes) addEdit(htmlFile, nameFrom, nameTo string) {
	if _, exists := c.edits[htmlFile]; !exists {
		c.edits[htmlFile] = make([]edit, 0, 1)
	}
	arr := c.edits[htmlFile]
	arr = append(arr, edit{
		fileNameFrom: nameFrom,
		fileNameTo:   nameTo,
		htmlFile:     htmlFile,
	})
	c.edits[htmlFile] = arr
}

func (c *changes) addError(htmlFile string, err error) {
	if _, exists := c.errors[htmlFile]; !exists {
		c.errors[htmlFile] = make([]editError, 0, 1)
	}
	arr := c.errors[htmlFile]
	arr = append(arr, editError{
		err:      err,
		htmlFile: htmlFile,
	})
	c.errors[htmlFile] = arr
}

func (c *changes) printChangesErrors() {
	if len(c.edits) == 0 {
		fmt.Println("No changes.")
	}
	for html, arr := range c.edits {
		if len(arr) == 0 {
			fmt.Println()
			continue
		}
		for _, edit := range arr {
			_, fFrom := filepath.Split(edit.fileNameFrom)
			_, fTo := filepath.Split(edit.fileNameTo)
			fmt.Printf("\n[%s] %s => %s", html, fFrom, fTo)
		}
	}

	for html, arr := range c.errors {
		if len(arr) == 0 {
			fmt.Println()
			continue
		}
		for _, edit := range arr {
			fmt.Printf("\n[%s] ERROR: %s", html, edit.err.Error())
		}
	}
}

func appendHashes(baseDir string) *changes {
	changes := &changes{
		edits:  make(map[string][]edit),
		errors: make(map[string][]editError),
	}

	htmlFilePaths, err := htmlFilePaths(baseDir)
	if err != nil {
		changes.addError("", err)
		return changes
	}

	var editJobs = []*job{}
	for _, filePath := range htmlFilePaths {
		b, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}
		addEditJobs(changes, &editJobs, filePath, string(b))
	}
	renameAll(changes, editJobs)
	return changes
}

func htmlFilePaths(baseDir string) ([]string, error) {
	var htmlFilePaths []string
	err := filepath.Walk(baseDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			split := strings.Split(info.Name(), ".")
			if len(split) > 0 {
				ext := split[len(split)-1]
				if ext == "html" || ext == "htm" {
					htmlFilePaths = append(htmlFilePaths, path)
				}
			}
			return nil
		})
	if err != nil {
		return nil, nil
	}
	return htmlFilePaths, nil
}

type job struct {
	fileNameWantToRename string
	filePathWantToRename string
	renameTo             string
	tagPath              string
	wholeTag             string
	htmlFile             string
}

type renameJob struct {
	pathFrom string
	pathTo   string
	htmlFile string
}

func renameAll(changes *changes, jobs []*job) {
	for _, job := range jobs {
		job.filePathWantToRename = filepath.Clean(job.filePathWantToRename)
	}

	renameJobs := make(map[string]renameJob)
	for _, job := range jobs {
		dir, _ := filepath.Split(job.filePathWantToRename)
		renameJobs[job.filePathWantToRename] = renameJob{
			pathFrom: job.filePathWantToRename,
			pathTo:   dir + job.renameTo,
			htmlFile: job.htmlFile,
		}
	} // only want to rename a file once, else we may attempt to rename an already renamed file, and it won't exist

	for _, job := range renameJobs {
		err := os.Rename(job.pathFrom, job.pathTo)
		if err != nil {
			changes.addError(job.htmlFile, err)
		}
	}

	// batch html edits
	for _, job := range jobs {
		fileContent, err := ioutil.ReadFile(job.htmlFile)
		if err != nil {
			changes.addError(job.htmlFile, err)
		}

		newFileContent := strings.ReplaceAll(string(fileContent), job.wholeTag, newTag(job))
		err = ioutil.WriteFile(job.htmlFile, []byte(newFileContent), 0644)
		if err != nil {
			changes.addError(job.htmlFile, err)
			continue
		}
		changes.addEdit(job.htmlFile, job.filePathWantToRename, job.renameTo)
	}
}

func newTag(j *job) string {
	newTag := strings.ReplaceAll(j.wholeTag, j.tagPath+j.fileNameWantToRename, j.tagPath+j.renameTo)
	return strings.ReplaceAll(j.wholeTag, j.wholeTag, newTag)
}

type tagInfo struct {
	tagType  string
	wholeTag string
	startTag int
}

func addEditJobs(editsErrors *changes, jobs *[]*job, htmlFilePath, fileContent string) {
	dir, _ := filepath.Split(htmlFilePath)
	tags := tagsFromHTML(fileContent)
	for _, ti := range tags {
		if ti.tagType == "script" {
			src, err := srcFilePath(ti.wholeTag)
			if err != nil && err.Error() == "src is empty" || httpPrefixed(src) {
				continue // normal for script tags to not have srcs
			}
			if err != nil {
				editsErrors.addError(htmlFilePath, err)
				continue
			}
			addJob(editsErrors, jobs, dir, src, htmlFilePath, ti.wholeTag)
		}
		if ti.tagType == "link" {
			href, err := hrefFilePath(ti.wholeTag)
			if httpPrefixed(href) {
				continue
			}
			if err != nil {
				if err.Error() != "href is empty" && err.Error() != "href is not css file" {
					editsErrors.addError(htmlFilePath, err)
				}
				continue
			}
			addJob(editsErrors, jobs, dir, href, htmlFilePath, ti.wholeTag)
		}
	}
}

func httpPrefixed(s string) bool {
	prefix := "http"
	if len(s) < len(prefix) {
		return false
	}
	if s[:4] == prefix {
		return true
	}
	return false
}

func addJob(changes *changes, jobs *[]*job, dir string, srcHref string, htmlFilePath string, wholeTag string) {
	hashedFileName, err := getHashedFileName(dir + srcHref) // change to rename file
	if err != nil {
		changes.addError(htmlFilePath, err)
		return
	}

	_, originalName := filepath.Split(dir + srcHref)
	tagLocalPath, _ := filepath.Split(srcHref)
	*jobs = append(*jobs, &job{
		fileNameWantToRename: originalName,
		filePathWantToRename: dir + srcHref,
		renameTo:             hashedFileName,
		tagPath:              tagLocalPath,
		htmlFile:             htmlFilePath,
		wholeTag:             wholeTag,
	})
}

// Renames file at filePath with a cache clobber certified hash.
// Returns a newly renamed filepath.
// Will remove the previous cc hash if it exists.
func getHashedFileName(filePath string) (string, error) {
	clean := filepath.Clean(filePath)
	b, err := ioutil.ReadFile(clean)
	if err != nil {
		return "", err
	}
	ccHash := "cc" + fmt.Sprint(hash(string(b))) // cc for CACHE CLOBBER
	_, fileName := filepath.Split(filePath)

	splitAtDash := strings.Split(fileName, "-")
	hashAndExt := splitAtDash[len(splitAtDash)-1]
	splitAtDot := strings.Split(hashAndExt, ".")
	possibleHash := splitAtDot[len(splitAtDot)-2]

	if isCCHash(possibleHash) {
		newFileName := strings.Replace(fileName, possibleHash, ccHash, 1)
		return newFileName, nil
	}

	ext := filepath.Ext(filePath)
	newFileName := fileName[:len(fileName)-len(ext)] + "-" + ccHash + ext
	return newFileName, nil
}

func isCCHash(s string) bool {
	if len(s) < 3 { //"cc#" is minimum
		return false
	}
	if s[:2] != "cc" {
		return false
	}
	return true
}

func hash(s string) uint32 {
	hasher := crc32.New(crc32.IEEETable)
	_, err := io.WriteString(hasher, s)
	if err != nil {
		fmt.Println(err)
	}
	return hasher.Sum32()
}

// Runs func op for each tag found in fileContent.
func tagsFromHTML(fileContent string) []tagInfo {
	tags := make([]tagInfo, 0, 0)
	insideTag := false
	readTagType := false // read = past tense

	startTag := 0
	currTagType := ""
	for i, char := range fileContent {
		if char == '<' {
			startTag = i
			insideTag = true
			continue
		}
		if char == '>' {
			wholeTag := fileContent[startTag : i+1]
			tags = append(tags, tagInfo{
				tagType:  currTagType,
				wholeTag: wholeTag,
				startTag: startTag,
			})
			currTagType = ""
			readTagType = false
			insideTag = false
			continue
		}
		if insideTag {
			if char == ' ' && !readTagType {
				currTagType = fileContent[startTag+1 : i] // startTag+1 cuts off the <
				readTagType = true
			}
			if !readTagType {
				currTagType = currTagType + string(char)
			}
		}
	}
	return tags
}

func hrefFilePath(wholeTag string) (string, error) {
	start := strings.Index(wholeTag, `href="`)
	start += len(`href="`)
	filePath := ""
	for i := start; i < len(wholeTag); i++ {
		if wholeTag[i] == '"' {
			filePath = wholeTag[start:i]
			break
		}
	}

	if filePath == "" {
		return "", errors.New("href is empty")
	}

	if !strings.Contains(filePath, ".css") {
		return "", errors.New("href is not css file")
	}
	return filePath, nil // cuts off href=" and "
}

func srcFilePath(wholeTag string) (string, error) {
	start := strings.Index(wholeTag, `src="`)
	start += len(`src="`)
	filePath := ""
	for i := start; i < len(wholeTag); i++ {
		if wholeTag[i] == '"' {
			filePath = wholeTag[start:i]
			break
		}
	}

	if filePath == "" {
		return "", errors.New("src is empty")
	}

	if !strings.Contains(filePath, ".js") {
		return "", errors.New("src is not js file")
	}
	return filePath, nil // cuts off src=" and "
}
