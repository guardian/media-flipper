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
