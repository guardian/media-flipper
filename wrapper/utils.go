package main

import (
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"
)

func RemoveExtension(from string) string {
	matcher := regexp.MustCompile("^(.*)\\.([^.]*)$")

	result := matcher.FindAllStringSubmatch(from, -1)
	if result == nil {
		return from
	} else {
		return result[0][1]
	}
}

/**
combine the input filename and a possible output path to make a full path to output filename
arguments:
- maybeOutPath - either an empty string "" or a directory name to output to. If non-empty, the directory portion of inPath is replaced
with this path, if empty then the directory portion of inPath is used
- inPath - full path to the incoming filename. This is used to determine the base of the filename and also the full path if maybeOutPath is empty
- suffix - portion to go after the filename with an _.  E.g. "thumb" -> myfilename_thumb.jpg
- xtn - file extension to be added.
*/
func GetOutputFilenameFull(maybeOutPath string, inPath string, suffix string, xtn string) string {
	var xtnStringWithDot string
	if strings.HasPrefix(xtn, ".") || xtn == "" {
		xtnStringWithDot = xtn
	} else {
		xtnStringWithDot = "." + xtn
	}

	origFileName := fmt.Sprintf("%s_%s%s", RemoveExtension(inPath), suffix, xtnStringWithDot)

	if maybeOutPath != "" {
		outFileName := path.Join(maybeOutPath, path.Base(origFileName))
		log.Printf("INFO: GetOutputFilename output path is %s from %s", outFileName, maybeOutPath)
		return outFileName
	} else {
		log.Printf("INFO: GetOutputFilename no provided output path, output is %s", origFileName)
		return origFileName
	}
}

func GetOutputFilenameThumb(maybeOutPath string, inPath string) string {
	return GetOutputFilenameFull(maybeOutPath, inPath, "thumb", ".jpg")
}

func GetOutputFileTransc(maybeOutPath string, inPath string, xtn string) string {
	return GetOutputFilenameFull(maybeOutPath, inPath, "transc", xtn)
}
