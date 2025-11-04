package main

import "testing"

func Test_SampleMethod(t *testing.T) {
	want := 10
	got := SampleMethod(4, 6)

	if want != got {
		t.Errorf("Expected %d, got %d", want, got)
	}
}
