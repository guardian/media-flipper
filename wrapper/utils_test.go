package main

import "testing"

func TestRemoveExtensionNormal(t *testing.T) {
	result := RemoveExtension("/path/to/somefile.ext")
	if result != "/path/to/somefile" {
		t.Error("Got unexpected result ", result)
	}
}

func TestRemoveExtensionNoExt(t *testing.T) {
	result := RemoveExtension("/path/to/somefile")
	if result != "/path/to/somefile" {
		t.Error("Got unexpected result ", result)
	}
}

func TestRemoveEmptyString(t *testing.T) {
	result := RemoveExtension("")
	if result != "" {
		t.Error("Got unexpected result ", result)
	}
}

func TestGetOutputFilenameFull(t *testing.T) {
	normalResult := GetOutputFilenameFull("/path/to/output", "/path/to/input/myfile.mkv", "blah", "jpg")
	if normalResult != "/path/to/output/myfile_blah.jpg" {
		t.Errorf("GetOutputfileNameFull returned unexpected value: got %s expected %s", normalResult, "/path/to/output/myfile_blah.jpg")
	}

	inPathResult := GetOutputFilenameFull("", "/path/to/input/myfile.mkv", "blah", "jpg")
	if inPathResult != "/path/to/input/myfile_blah.jpg" {
		t.Errorf("GetOutputFilenameFull returned unexpected value: got %s expected %s", inPathResult, "/path/to/input/myfile_blah.jpg")
	}

	noXtnResult := GetOutputFilenameFull("", "/path/to/input/myfile.mkv", "blah", "")
	if noXtnResult != "/path/to/input/myfile_blah" {
		t.Errorf("GetOutputFilenameFile returned unexpected value: got %s expected %s", noXtnResult, "/path/to/input/myfile_blah")
	}
}
