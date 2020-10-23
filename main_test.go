package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendHashes(t *testing.T) {
	defer func() {
		//err := os.RemoveAll("./test")
		//if err != nil {
		//	t.Fatal(err)
		//}
	}()
	createTestDirFiles(t)

	var filePaths = []string{
		"test/lame.js",
		"test/cooler.js",
		"test/cool.js",
		"test/already-hashed-cc123.js",
		"test/weird-ccna-name.js",
		"test/weird-ccna-name-already-hashed-cc123.js",
		"test/assets/big.js",
		"test/assets/bloat.js",
		"test/styles.css",
		"test/more-styles.css",
		"test/assets/pretty-styles.css",
		"test/assets/ugly-styles.css",
	}

	// Using fileContents to see if all renamed files were the original ones, checked in checkNeccesaryFilesChanges
	var fileContents = make(map[string]struct{})
	for _, fp := range filePaths {
		b, err := ioutil.ReadFile(fp)
		if err != nil {
			t.Error(err)
		}
		fileContents[string(b)] = struct{}{}
	}

	appendHashes()

	filesInPaths := []string{}
	err := filepath.Walk("./test",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			filesInPaths = append(filesInPaths, path)
			return nil
		})
	if err != nil {
		t.Error(err)
	}

	newNames := make(map[string]bool) // [fileName]existsInHTML, existsInHTML checked in checkHTMLEdits
	newNamesPaths := []string{}
	for _, f := range filesInPaths {
		for _, ff := range filePaths {
			a := filepath.Clean(removeCCHash(f))
			b := filepath.Clean(removeCCHash(ff))
			if a == b {
				newNamesPaths = append(newNamesPaths, f)
				_, name := filepath.Split(f)
				newNames[name] = false
			}
		}
	}

	checkNeccesaryFilesChanges(t, newNamesPaths, fileContents)

	// Check if HTML was edited with renamed (newNames) files
	checkHTMLEdits(t, filesInPaths, newNames)
}

func checkNeccesaryFilesChanges(t *testing.T, newNamesPaths []string, fileContents map[string]struct{}) {
	var newNamesFileContents = make(map[string]struct{})
	for _, f := range newNamesPaths {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			t.Error(err)
		}
		newNamesFileContents[string(b)] = struct{}{}
	}

	for nfc, _ := range newNamesFileContents {
		cont, exists := fileContents[nfc]
		if exists {
			delete(fileContents, nfc)
			continue
		}
		t.Error("Did not find new file with matching content", cont)
	}
}

func checkHTMLEdits(t *testing.T, filesInPaths []string, newNames map[string]bool) {
	htmlFileContents := []string{}
	for _, f := range filesInPaths {
		if filepath.Ext(f) == ".html" {
			cont, err := ioutil.ReadFile(f)
			if err != nil {
				t.Error(err)
			}
			htmlFileContents = append(htmlFileContents, string(cont))
		}
	}

	for _, fc := range htmlFileContents {
		for newName, _ := range newNames {
			if strings.Contains(fc, newName) {
				newNames[newName] = true
			}
		}
	}

	for newName, foundInHTML := range newNames {
		if !foundInHTML {
			t.Error(newName, "was not found in HTML")
		}
	}
}

func TestRemoveCCHash(t *testing.T) {
	in := "file-cc234234234.js"
	expected := "file.js"
	if actual := removeCCHash(in); actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}

	in = "file-cc0.js"
	expected = "file.js"
	if actual := removeCCHash(in); actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}

	in = "file-ccna-cc123.js"
	expected = "file-ccna.js"
	if actual := removeCCHash(in); actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}

	in = "file-ccna-cc0.js"
	expected = "file-ccna.js"
	if actual := removeCCHash(in); actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}

	in = "file.js"
	expected = "file.js"
	if actual := removeCCHash(in); actual != expected {
		t.Errorf("expected %s, actual %s", expected, actual)
	}
}

func removeCCHash(s string) string {
	i := strings.LastIndex(s, "-cc")
	if i == -1 {
		return s
	}
	return s[:i] + filepath.Ext(s)
}

func createTestDirFiles(t *testing.T) {
	err := os.RemoveAll("./test")
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll("./test/assets", 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/already-hashed-cc123.js", []byte(`console.log("cool and AMAZING")`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/weird-ccna-name.js", []byte(`console.log("cool and FANTASTIC")`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/weird-ccna-name-already-hashed-cc123.js", []byte(`console.log("cool and alright alright alright")`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/cool.js", []byte(`console.log("cool and good")`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/cooler.js", []byte(`console.log("cooler and good")`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/lame.js", []byte(`console.log("lame and good")`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/styles.css", []byte(`h1{font-size:12px;}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/more-styles.css", []byte(`body{background-color:red;}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/assets/pretty-styles.css", []byte(`h2{font-size:20px;}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/assets/ugly-styles.css", []byte(`h3{font-size:30px;}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/assets/big.js", []byte(`console.log("big and good")`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("./test/assets/bloat.js", []byte(`console.log("bloat and good")`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/index.html", []byte(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<title>Title</title>
			<link rel="stylesheet" href="styles.css">
			<link rel="stylesheet" href="more-styles.css">
			<script src="cool.js"></script>
			<script src="cool.js"></script>
			<script src="./cooler.js"></script>
			<script src="already-hashed-cc123.js"></script>

			<script src="./weird-ccna-name.js"></script>
			<script src="./weird-ccna-name-already-hashed-cc123.js"></script>

			<script src="assets/bloat.js"></script>
			<script src="./assets/big.js"></script>
			<link rel="stylesheet" href="assets/pretty-styles.css">
			<link rel="stylesheet" href="./assets/ugly-styles.css">
		</head>
		<body>
			<h1>Fear</h1>
			<h2>The</h2>
			<h3>Cache Clobber</h3>
		</body>
		</html>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile("./test/assets/markup.html", []byte(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<title>Title</title>
			<script src="../lame.js"></script>
			<script src="../cooler.js"></script>
			<script src="../cool.js"></script>
			<link rel="stylesheet" href="./pretty-styles.css">
			<link rel="stylesheet" href="../assets/ugly-styles.css">
		</head>
		<body>
			<h1>Fear</h1>
			<h2>The</h2>
			<h3>Cache Clobber</h3>
		</body>
		</html>`), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHrefFilePath(t *testing.T) {
	nonErrorTests := []struct {
		in       string // input
		expected string // expected result
	}{
		{`<script src="lame.js"></script>`, `lame.js`},
		{`<script src="../lame.js"></script>`, `../lame.js`},
		{`<script type="text/javascript" src="../lame.js"></script>`, `../lame.js`},
		{`<script src="./big.js"></script>`, `./big.js`},
	}

	for _, tt := range nonErrorTests {
		actual, err := srcFilePath(tt.in)
		if actual != tt.expected {
			t.Errorf("hrefFilePath(%s): expected %s, actual %s", tt.in, tt.expected, actual)
		}
		if err != nil {
			t.Errorf("hrefFilePath(%s): errored %s", tt.in, err)
		}
	}

	errorTests := []struct {
		in       string // input
		expected error  // expected result
	}{
		{`<script src=""></script>`, errors.New("src is empty")},
		{`<script src="  "></script>`, errors.New("src is not js file")},
		{`<script type="text/javascript" src="../bad.php"></script>`, errors.New("src is not js file")},
		{`<script src="./big.css"></script>`, errors.New("src is not js file")},
	}

	for _, tt := range errorTests {
		_, err := srcFilePath(tt.in)
		if err.Error() != tt.expected.Error() {
			t.Errorf("hrefFilePath(%s): expected err %s, actual err %s", tt.in, tt.expected, err)
		}
	}
}

func TestSrcFilePath(t *testing.T) {
	nonErrorTests := []struct {
		in       string // input
		expected string // expected result
	}{
		{`<link href="lame.css"></link>`, `lame.css`},
		{`<link href="../lame.css"></link>`, `../lame.css`},
		{`<link href="../lame.css"></link>`, `../lame.css`},
		{`<link href="./big.css"></link>`, `./big.css`},
	}

	for _, tt := range nonErrorTests {
		actual, err := hrefFilePath(tt.in)
		if actual != tt.expected {
			t.Errorf("hrefFilePath(%s): expected %s, actual %s", tt.in, tt.expected, actual)
		}
		if err != nil {
			t.Errorf("hrefFilePath(%s): errored %s", tt.in, err)
		}
	}

	errorTests := []struct {
		in       string // input
		expected error  // expected result
	}{
		{`<link href=""></link>`, errors.New("href is empty")},
		{`<link href="  "></link>`, errors.New("href is not css file")},
		{`<link href="../bad.php"></link>`, errors.New("href is not css file")},
		{`<link href="./big.js"></link>`, errors.New("href is not css file")},
	}

	for _, tt := range errorTests {
		_, err := hrefFilePath(tt.in)
		if err != nil && err.Error() != tt.expected.Error() {
			t.Errorf("hrefFilePath(%s): expected err %s, actual err %s", tt.in, tt.expected, err)
		}
		if err == nil {
			t.Errorf("hrefFilePath(%s): expected err %s, actual err %s", tt.in, tt.expected, err)
		}
	}
}

func TestIsCCHash(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "1", args: args{"cc234"}, want: true},
		{name: "2", args: args{"cc346363633535353"}, want: true},
		{name: "3", args: args{"cc3brtbrtyetwtwerwerwetyuiuii"}, want: true},
		{name: "4", args: args{"cckwrwrwrwr"}, want: true},
		{name: "5", args: args{"cc62626252525890862345678"}, want: true},
		{name: "6", args: args{"cc1"}, want: true},
		{name: "7", args: args{"cc0"}, want: true},
		{name: "8", args: args{"cc9"}, want: true},
		{name: "9", args: args{"cc"}, want: false},
		{name: "10", args: args{"-blublubblub"}, want: false},
		{name: "11", args: args{"ccblubblub"}, want: true},
		{name: "12", args: args{"--cc74747"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCCHash(tt.args.s); got != tt.want {
				t.Errorf("isCCHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
