package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendHashes(t *testing.T) {
	defer func() {
		//cleanTestDirectory(t)
	}()
	cleanTestDirectory(t)
	createTestDirFiles(t)

	// all files that can be changed, kept for ease of use
	var allFiles = []string{
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

	cleanTestDirectory(t)
	createTestDirFiles(t)
	baseDir := "./test"
	{
		expectedChangedFiles := allFiles
		changes := appendHashes(baseDir)
		leftoverChanges, leftoverExpected := deleteMatches(allChangesToOneSlice(changes), expectedChangedFiles)
		for _, changeNotDone := range leftoverExpected {
			t.Error("expected file to change, but it did not:", changeNotDone)
		}
		for _, changeDone := range leftoverChanges {
			t.Error("did not specify file to change, but it did:", changeDone)
		}
		if len(changes.errors) != 0 {
			for _, v := range changes.errors {
				for _, vv := range v {
					t.Error("encountered error", vv.err)
				}
			}
		}
	}

	cleanTestDirectory(t)
	createTestDirFiles(t)
	baseDir = "./"
	{
		expectedChangedFiles := allFiles
		changes := appendHashes(baseDir)
		leftoverChanges, leftoverExpected := deleteMatches(allChangesToOneSlice(changes), expectedChangedFiles)
		for _, changeNotDone := range leftoverExpected {
			t.Error("expected file to change, but it did not:", changeNotDone)
		}
		for _, changeDone := range leftoverChanges {
			t.Error("did not specify file to change, but it did:", changeDone)
		}
		if len(changes.errors) != 0 {
			for _, v := range changes.errors {
				for _, vv := range v {
					t.Error("encountered error", vv.err)
				}
			}
		}
	}

	cleanTestDirectory(t)
	createTestDirFiles(t)
	baseDir = "./test/assets"
	{
		expectedChangedFiles := []string{
			"lame.js",
			"cooler.js",
			"cool.js",
			"pretty-styles.css",
			"ugly-styles.css",
		}
		changes := appendHashes(baseDir)

		leftoverChanges, leftoverExpected := deleteMatches(allChangesToOneSlice(changes), expectedChangedFiles)
		for _, changeNotDone := range leftoverExpected {
			t.Error("expected file to change, but it did not:", changeNotDone)
		}
		for _, changeDone := range leftoverChanges {
			t.Error("did not specify file to change, but it did:", changeDone)
		}
		if len(changes.errors) != 0 {
			for _, v := range changes.errors {
				for _, vv := range v {
					t.Error("encountered error", vv.err)
				}
			}
		}
	}
}

func allChangesToOneSlice(changes *changes) []edit {
	edits := []edit{}
	for _, html := range changes.edits {
		for _, g := range html {
			edits = append(edits, g)
		}
	}
	return edits
}

func deleteMatches(a []edit, b []string) ([]string, []string) {
	bMap := make(map[string]struct{})
	for _, v := range b {
		bMap[v] = struct{}{}
	}
	bMapCopy := make(map[string]struct{})
	for k, v := range bMap {
		bMapCopy[k] = v
	}

	aMap := make(map[edit]struct{})
	for _, v := range a {
		aMap[v] = struct{}{}
	}
	aMapCopy := make(map[edit]struct{})
	for k, v := range aMap {
		aMapCopy[k] = v
	}

	// reduce aMap
	for chg, _ := range aMap {
		for b := range bMapCopy {
			_, chgFName := filepath.Split(chg.fileNameFrom)
			_, bFName := filepath.Split(b)
			if chgFName == bFName {
				delete(aMap, chg)
				delete(bMap, b)
			}
		}
	}

	// reduce bMap
	for chg, _ := range aMapCopy {
		for b := range bMap {
			_, chgFName := filepath.Split(chg.fileNameFrom)
			_, bFName := filepath.Split(b)
			if chgFName == bFName {
				delete(aMap, chg)
				delete(bMap, b)
			}
		}
	}

	shouldHaveChanged := []string{}
	for v, _ := range bMap {
		shouldHaveChanged = append(shouldHaveChanged, v)
	}

	changedButWasNotToldToChange := []string{}
	for v, _ := range aMap {
		changedButWasNotToldToChange = append(changedButWasNotToldToChange, v.fileNameFrom)
	}
	return changedButWasNotToldToChange, shouldHaveChanged
}

func cleanTestDirectory(t *testing.T) {
	err := os.RemoveAll("./test")
	if err != nil {
		t.Fatal(err)
	}
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
			<script> console.log("I belong to no one.") </script>
			<script src="https://code.jquery.com/jquery-3.5.1.min.js"></script>
			<script src="./cooler.js"></script>
			<script src="already-hashed-cc123.js"></script>

			<script src="./weird-ccna-name.js"></script>
			<script src="./weird-ccna-name-already-hashed-cc123.js"></script>

			<link href="https://fonts.googleapis.com/css?family=Bowlby+One+SC|Cabin&display=swap" rel="stylesheet">
			<link rel='icon' type='image/png' href='./favicon.png'>
			<link rel='stylesheet' href='./global.css'>

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

func TestSrcFilePath(t *testing.T) {
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

func TestHrefFilePath(t *testing.T) {
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
		{`<link href="h" rel="stylesheet">`, errors.New("href is not css file")},
		{`<link href="" rel="stylesheet">`, errors.New("href is empty")},
		{`<link href="https://fonts.googleapis.com/css?family=Bowlby+One+SC|Cabin&display=swap" rel="stylesheet">`, errors.New("href is not css file")},
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

func TestHttpPrefixed(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "", args: args{"https://code.jquery.com/jquery-3.5.1.min.js"}, want: true},
		{name: "", args: args{"http://lodash.com"}, want: true},
		{name: "", args: args{"http"}, want: true},
		{name: "", args: args{"ht"}, want: false},
		{name: "", args: args{"h"}, want: false},
		{name: "", args: args{""}, want: false},
		{name: "", args: args{"yay.js"}, want: false},
		{name: "", args: args{"./test/deep/down/file.js"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httpPrefixed(tt.args.s); got != tt.want {
				t.Errorf("httpPrefixed() = %v, want %v", got, tt.want)
			}
		})
	}
}
