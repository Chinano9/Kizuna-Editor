package audio

import "testing"

func TestStopRecordingWithInfoRequiresMatchingRecordingType(t *testing.T) {
	recorder := NewAudioRecorder()
	recorder.isRecording = true
	recorder.isQuickIdea = true
	recorder.SongID = 0
	recorder.TrackName = "Quick Idea"

	regularRecording := false
	if _, _, _, _, _, err := recorder.StopRecordingWithInfo(&regularRecording); err == nil {
		t.Fatal("expected error when stopping a quick idea as a project take")
	}
	if !recorder.IsRecording() {
		t.Fatal("expected mismatched stop request to leave recording active")
	}

	quickIdea := true
	_, _, _, _, info, err := recorder.StopRecordingWithInfo(&quickIdea)
	if err != nil {
		t.Fatalf("failed to stop quick idea recording: %v", err)
	}
	if !info.QuickIdea || info.SongID != 0 || info.TrackName != "Quick Idea" {
		t.Errorf("unexpected recording metadata: %+v", info)
	}
	if recorder.IsRecording() {
		t.Fatal("expected recorder to stop")
	}
}
