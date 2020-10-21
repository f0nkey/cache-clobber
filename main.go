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
	appendHashes()
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

// todo: finish changes
var fileChanges = make(changes)
// [fileName.html]changes
type changes map[string]string

func (c changes) addChange(htmlFile, change string){

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
		}] = struct{}{}
	} // only want to rename a file once, else we may attempt to rename an already renamed file, and it won't exist

	for job, _ := range renameJobs {
		err := os.Rename(job.pathFrom, job.pathTo)
		if err != nil {
			fmt.Println(err)
			//return "", err
			// stop any that reference the file at the specific point
		}
	}

	// batch html edits
	for _, job := range jobs {
		fileContent,err := ioutil.ReadFile(job.htmlFile)
		if err != nil {
			fileChanges.addChange(job.htmlFile,fmt.Sprintf("\nERROR for tag: %s, %s", job.filePathWantToRename, err.Error()))
		}
		newFileContent := strings.ReplaceAll(string(fileContent),job.tagPath+job.fileNameWantToRename,job.tagPath+job.renameTo)
		err = ioutil.WriteFile(job.htmlFile, []byte(newFileContent), 0644)
		if err != nil {
			fileChanges.addChange(job.htmlFile,fmt.Sprintf("\nERROR for tag: %s, %s", job.filePathWantToRename, err.Error()))
		}
	}

}

// Edit the .html, files at src attributes, and files at href attributes by appending crc-32 hashes.
func editFileLinkSrc(filePath, fileContent string) {
	//fileChanges := fmt.Sprintf("\nChanges for %s:", filePath)
	//fileChangeErrors := fmt.Sprintf("\nErrors for %s:", filePath)
	dir, _ := filepath.Split(filePath)
	operateOnScannedTags(fileContent, func(tagType, wholeTag string, startTag int) {
		if tagType == "script" {
			src, err := srcFilePath(wholeTag)
			if err != nil && err.Error() == "src is empty"{
				return // normal for script tags to not have srcs
			}
			if err != nil {
				fmt.Println(err)
				//fileChangeErrors = fileChangeErrors + fmt.Sprintf("\nERROR for tag: %s, %s", wholeTag, err.Error())
			}

			hashedFileName, err := getHashedFileName(dir+src) // change to rename file
			if err != nil {
				fmt.Println(err)
				//fileChanges.addChange(job.htmlFile,fmt.Sprintf("\nERROR for tag: %s, %s", job.wholeTag, err.Error()))
			}

			_, originalName := filepath.Split(dir+src)
			tagLocalPath,_ := filepath.Split(src)
			editJobs = append(editJobs, &job{
				fileNameWantToRename: originalName,
				filePathWantToRename: dir+src,
				renameTo:             hashedFileName,
				tagPath:              tagLocalPath,
				htmlFile:             filePath,
				wholeTag:             wholeTag,
			})
		}
		if tagType == "link" {
			// todo: make work with link tags
		}
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
	dir, fileName := filepath.Split(filePath)

	splitAtDash := strings.Split(fileName,"-")
	hashAndExt := splitAtDash[len(splitAtDash)-1]
	splitAtDot := strings.Split(hashAndExt,".")
	possibleHash := splitAtDot[len(splitAtDot)-2]

	if isCCHash(possibleHash) {
		//todo: delete the hash, place hash in between the name and .js/.css
		newFileName := strings.Replace(fileName,possibleHash,"-"+ccHash,1)
		err = os.Rename(dir + fileName, dir + newFileName)
		if err != nil {
			return "", err
		}
		return newFileName, nil
	}

	//todo: place hash in between the name and .js/.css
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
