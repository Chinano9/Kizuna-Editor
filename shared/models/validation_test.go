package models

import "testing"

func TestIsValidTimeSignature(t *testing.T) {
	validSignatures := []string{"4/4", "3/4", "2/4", "6/8", "12/8", "7/8", "5/4", "2/2", "9/16"}
	for _, sig := range validSignatures {
		if !IsValidTimeSignature(sig) {
			t.Errorf("expected '%s' to be valid time signature", sig)
		}
	}

	invalidSignatures := []string{"", "   ", "4-4", "4/3", "4/5", "abc", "4/", "/4", "4/0"}
	for _, sig := range invalidSignatures {
		if IsValidTimeSignature(sig) {
			t.Errorf("expected '%s' to be invalid time signature", sig)
		}
	}
}

func TestIsValidClef(t *testing.T) {
	validClefs := []string{"treble", "bass", "percussion", "alto", "tenor", "TREBLE", " Bass "}
	for _, clef := range validClefs {
		if !IsValidClef(clef) {
			t.Errorf("expected '%s' to be valid clef", clef)
		}
	}

	invalidClefs := []string{"", "guitar", "piano", "random"}
	for _, clef := range invalidClefs {
		if IsValidClef(clef) {
			t.Errorf("expected '%s' to be invalid clef", clef)
		}
	}
}
