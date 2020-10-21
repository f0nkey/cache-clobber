package main

import (
	"errors"
	"io/ioutil"
	"os"
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
	appendHashes()
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
			<script src="./cooler.js"></script>
		
		
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
		expected error // expected result
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
		expected error // expected result
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
		if err == nil  {
			t.Errorf("hrefFilePath(%s): expected err %s, actual err %s", tt.in, tt.expected, err)
		}
	}
}
