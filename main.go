package main

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	chgAttempts = changeAttempts{
		changes: make(map[string]string),
		errors:   make(map[string]string),
	}
	appendHashes()
	chgAttempts.printChangesErrors()
}

type changeAttempts struct {
	changes map[string]string // [fileName.html]changeAttempts
	errors map[string]string // [fileName.html]changeAttempts
}

var chgAttempts changeAttempts

func (c *changeAttempts) addChange(htmlFile, nameFrom, nameTo string) {
	if _, exists := c.changes[htmlFile]; !exists {
		c.changes[htmlFile] = "Changes for " + htmlFile + ":"
	}
	c.changes[htmlFile] = c.changes[htmlFile] + fmt.Sprintf("\n%s => %s", nameFrom, nameTo)
}

func (c *changeAttempts) addError(htmlFile, err string) {
	if _, exists := c.errors[htmlFile]; !exists {
		c.errors[htmlFile] = "Errors for " + htmlFile + ":"
	}
	c.errors[htmlFile] = c.errors[htmlFile] + fmt.Sprintf("\nERROR:%s", err)
}

func (c *changeAttempts) printChangesErrors() {
	for htmlChg, changes := range c.changes {
		thisErrs := ""
		// grab the errors associated with the curr htmlChg
		for htmlErr, errs := range c.errors {
			if htmlChg == htmlErr {
				thisErrs = errs
			}
		}
		fmt.Println(changes)
		fmt.Println(thisErrs)
	}
}


func appendHashes() {
	htmlFilePaths, err := htmlFilePaths()
	if err != nil {
		log.Fatal(err)
	}

	for _, filePath := range htmlFilePaths {
		b, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}
		editFileLinkSrc(filePath, string(b))
	}
	renameAll(editJobs)
}

func htmlFilePaths() ([]string, error) {
	var htmlFilePaths []string
	err := filepath.Walk(".",
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
	return htmlFilePaths,  nil
}

var editJobs []*job

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
	pathTo string
	htmlFile string
}

func renameAll(jobs []*job) {
	for _, job := range jobs {
		job.filePathWantToRename = filepath.Clean(job.filePathWantToRename)
	}

	renameJobs := make(map[renameJob]struct{})
	for _, job := range jobs {
		dir, _ := filepath.Split(job.filePathWantToRename)
		renameJobs[renameJob{
			pathFrom: job.filePathWantToRename,
			pathTo:   dir + job.renameTo,
			htmlFile: job.htmlFile,
		}] = struct{}{}
	} // only want to rename a file once, else we may attempt to rename an already renamed file, and it won't exist

	for job, _ := range renameJobs {
		err := os.Rename(job.pathFrom, job.pathTo)
		if err != nil {
			chgAttempts.addError(job.htmlFile, err.Error())
		}
	}

	// batch html edits
	for _, job := range jobs {
		fileContent,err := ioutil.ReadFile(job.htmlFile)
		if err != nil {
			chgAttempts.addError(job.htmlFile, err.Error())
		}

		newFileContent := strings.ReplaceAll(string(fileContent), job.wholeTag, newTag(job))
		err = ioutil.WriteFile(job.htmlFile, []byte(newFileContent), 0644)
		if err != nil {
			chgAttempts.addError(job.htmlFile, err.Error())
			continue
		}
		chgAttempts.addChange(job.htmlFile, job.filePathWantToRename, job.renameTo)
	}
}

func newTag(j *job) string {
	newTag := strings.ReplaceAll(j.wholeTag, j.tagPath+j.fileNameWantToRename, j.tagPath+j.renameTo)
	return strings.ReplaceAll(j.wholeTag, j.wholeTag, newTag)
}

// Edit the .html, files at src attributes, and files at href attributes by appending crc-32 hashes.
func editFileLinkSrc(htmlFilePath, fileContent string) {
	dir, _ := filepath.Split(htmlFilePath)
	operateOnScannedTags(fileContent, func(tagType, wholeTag string, startTag int) {
		if tagType == "script" {
			src, err := srcFilePath(wholeTag)
			if err != nil && err.Error() == "src is empty"{
				return // normal for script tags to not have srcs
			}
			if err != nil {
				chgAttempts.addError(htmlFilePath, err.Error())
				return
			}

			addJob(dir, src, htmlFilePath, wholeTag)
		}
		if tagType == "link" {
			href, err := hrefFilePath(wholeTag)
			if err != nil {
				chgAttempts.addError(htmlFilePath, err.Error())
				return
			}
			addJob(dir, href, htmlFilePath, wholeTag)
		}
	})

}

func addJob(dir string, srcHref string, htmlFilePath string, wholeTag string) {
	hashedFileName, err := getHashedFileName(dir + srcHref) // change to rename file
	if err != nil {
		chgAttempts.addError(htmlFilePath, err.Error())
		return
	}

	_, originalName := filepath.Split(dir + srcHref)
	tagLocalPath, _ := filepath.Split(srcHref)
	editJobs = append(editJobs, &job{
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
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	ccHash := "cc" + fmt.Sprint(hash(string(b))) // cc for CACHE CLOBBER
	_, fileName := filepath.Split(filePath)

	splitAtDash := strings.Split(fileName,"-")
	hashAndExt := splitAtDash[len(splitAtDash)-1]
	splitAtDot := strings.Split(hashAndExt,".")
	possibleHash := splitAtDot[len(splitAtDot)-2]

	if isCCHash(possibleHash) {
		newFileName := strings.Replace(fileName,possibleHash,ccHash,1)
		return newFileName, nil
	}

	ext := filepath.Ext(filePath)
	newFileName := fileName[:len(fileName)-len(ext)] + "-"+ccHash + ext
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
	_, err := io.WriteString(hasher,s)
	if err != nil {
		fmt.Println(err)
	}
	return hasher.Sum32()
}

// Runs func op for each tag found in fileContent.
func operateOnScannedTags(fileContent string, op func(tagType, wholeTag string, startTag int)) {
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
			wholeTag := fileContent[startTag:i+1]
			op(currTagType, wholeTag, startTag)
			currTagType = ""
			readTagType = false
			insideTag = false
			continue
		}
		if insideTag {
			if char == ' ' && !readTagType {
				currTagType = fileContent[startTag+1:i] // startTag+1 cuts off the <
				readTagType = true
			}
			if !readTagType {
				currTagType = currTagType + string(char)
			}
		}
	}
}

func hrefFilePath(wholeTag string) (string, error) {
	r, err := regexp.Compile(`(href=".+")`)
	if err != nil {
		log.Fatal(err)
	}

	filePath := string(r.Find([]byte(wholeTag)))
	if filePath == "" {
		return "", errors.New("href is empty")
	}

	if !strings.Contains(filePath, ".css") {
		return "", errors.New("href is not css file")
	}
	return filePath[6:len(filePath)-1], nil // cuts off src=" and "
}

func srcFilePath(wholeTag string) (string, error) {
	r, err := regexp.Compile(`(src=".+")`)
	if err != nil {
		log.Fatal(err)
	}

	filePath := string(r.Find([]byte(wholeTag)))
	if filePath == "" {
		return "", errors.New("src is empty")
	}

	if !strings.Contains(filePath, ".js") {
		return "", errors.New("src is not js file")
	}
	return filePath[5:len(filePath)-1], nil // cuts off src=" and "
}
