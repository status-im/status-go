# goxpp

[![Build Status](https://travis-ci.org/mmcdole/goxpp.svg?branch=master)](https://travis-ci.org/mmcdole/goxpp) [![Coverage Status](https://coveralls.io/repos/github/mmcdole/goxpp/badge.svg?branch=master)](https://coveralls.io/github/mmcdole/goxpp?branch=master) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org)
[![GoDoc](https://godoc.org/github.com/mmcdole/goxpp?status.svg)](https://godoc.org/github.com/mmcdole/goxpp)

The `goxpp` library, inspired by [Java's XMLPullParser](http://www.xmlpull.org/v1/download/unpacked/doc/quick_intro.html), is a lightweight wrapper for Go's standard XML Decoder, tailored for developers who need fine-grained control over XML parsing. Unlike simple unmarshaling of entire documents, this library excels in scenarios requiring manual navigation and consumption of XML elements. It provides a pull parser approach with convenience methods for effortlessly consuming whole tags, skipping elements, and more, granting a level of flexibility and control beyond what Go's standard XML decode method offers.

## Overview

To begin parsing a XML document using `goxpp` you must pass it an `io.Reader` object for your document:

```go
file, err := os.Open("path/file.xml")
parser := xpp.NewXMLPullParser(file, false, charset.NewReader)
```

The `goxpp` library decodes documents into a series of token objects:

| Token Name                       |
|----------------------------------|
| 	StartDocument                  |
| 	EndDocument                    |
| 	StartTag                       |
| 	EndTag                         |
| 	Text                           |
| 	Comment                        |
| 	ProcessingInstruction          |
| 	Directive                      |
| 	IgnorableWhitespace            |

You will always start at the `StartDocument` token and can use the following functions to walk through a document:

| Function Name                    | Description                           |
|----------------------------------|---------------------------------------|
| 	 Next()                        | Advance to the next `Text`, `StartTag`, `EndTag`, `EndDocument` token.<br>Note: skips `Comment`, `Directive` and `ProcessingInstruction` |
| 	NextToken()                    | Advance to the next token regardless of type.                                                                |
| 	NextText()                     | Advance to the next `Text` token.                                                                |
| 	Skip()                         | Skip the next token.   |
| 	DecodeElement(v interface{})   | Decode an entire element from the current tag into a struct.<br>Note: must be at a `StartTag` token |



This project is licensed under the [MIT License](https://raw.githubusercontent.com/mmcdole/goxpp/master/LICENSE)

