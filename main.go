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

// appended to as changes/errors are made
var fileChanges string = "FILE CHANGES"
var fileChangeErrors string = "FILE CHANGE ERRORS"

// Edit the .html, files at src attributes, and files at href attributes by appending crc-32 hashes.
func editFileLinkSrc(filePath, fileContent string) {
	fileChanges = fileChanges + fmt.Sprintf("\nChanges for %s:", filePath)
	operateOnScannedTags(fileContent, func(tagType, wholeTag string, startTag int) {
		if tagType == "script" {
			src, err := srcFilePath(wholeTag)
			if err != nil && err.Error() == "src is empty"{ // normal for script tags to not have srcs
				return
			}
			if err != nil {
				fileChangeErrors = fileChangeErrors + fmt.Sprintf("\nERROR for tag: %s, %s", wholeTag, err.Error())
			}

			renamedPath, err := renameWithHash(src)
			if err != nil {
				fileChangeErrors = fileChangeErrors + fmt.Sprintf("\nERROR for tag: %s, %s", wholeTag, err.Error())
			}
			fileChanges = fileChanges + fmt.Sprintf("\n%s => %s", src, renamedPath)

			newFileContent := strings.Replace(fileContent,src,renamedPath,1)
			err = ioutil.WriteFile(filePath, []byte(newFileContent), 0644)
			if err != nil {
				fileChangeErrors = fileChangeErrors + fmt.Sprintf("\nERROR for tag: %s, %s", wholeTag, err.Error())
			}

		}
		if tagType == "link" {
			//href, err := hrefFilePath(wholeTag)
			//if err != nil {
			//	fileChangeErrors = fileChangeErrors + fmt.Sprintf("\nERROR for tag: %s, %s", wholeTag, err.Error())
			//}
		}
	})
}

// Renames file at filePath with a cache clobber certified hash.
// Returns a newly renamed filepath with a cache clobber certified hash.
// Will remove the previous cc hash if it exists.
func renameWithHash(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	ccHash := "cc" + fmt.Sprint(hash(string(b))) // cc for CACHE CLOBBER
	dir, fileName := filepath.Split(filePath)

	split := strings.Split(fileName,"-")
	possibleHash := split[len(split)]
	if isCCHash(possibleHash) {
		newFileName := strings.Replace(fileName,possibleHash,"-"+ccHash,1)
		err = os.Rename(dir + fileName, dir + newFileName)
		if err != nil {
			return "", err
		}
		return dir+newFileName, nil
	}

	newFileName := fileName + "-"+ccHash
	err = os.Rename(dir + fileName, dir + newFileName)
	if err != nil {
		return "", err
	}
	return dir+newFileName, nil
}

func isCCHash(s string) bool {
	if len(s) < 3 { //"cc#" is minimum
		return false
	}

}

// Returns
func hashName(s string) {

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
	// regex alternative to find tags <((?=!\-\-)!\-\-[\s\S]*\-\-|((?=\?)\?[\s\S]*\?|((?=\/)\/[^.\-\d][^\/\]'"[!#$%&()*+,;<=>?@^`{|}~ ]*|[^.\-\d][^\/\]'"[!#$%&()*+,;<=>?@^`{|}~ ]*(?:\s[^.\-\d][^\/\]'"[!#$%&()*+,;<=>?@^`{|}~ ]*(?:=(?:"[^"]*"|'[^']*'|[^'"<\s]*))?)*)\s?\/?))>
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
				currTagType = fileContent[startTag+1:i+1] // startTag+1 cuts off the <
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
