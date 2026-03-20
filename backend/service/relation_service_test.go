package service

import (
	"strings"
	"testing"
	"welink/backend/model"
)

func TestConfidenceWithReason_ConsidersSampleAndFreshness(t *testing.T) {
	highSampleRecent := &relationProfile{
		TotalMessages:        160,
		ReplySamples:         make([]float64, 12),
		TotalSessions:        20,
		DaysSinceLastContact: 8,
	}
	highRecentConfidence, _ := confidenceWithReason(highSampleRecent, 15)

	lowSampleRecent := &relationProfile{
		TotalMessages:        14,
		ReplySamples:         []float64{120},
		TotalSessions:        2,
		DaysSinceLastContact: 8,
	}
	lowRecentConfidence, _ := confidenceWithReason(lowSampleRecent, 15)

	lowSampleStale := &relationProfile{
		TotalMessages:        14,
		ReplySamples:         []float64{120},
		TotalSessions:        2,
		DaysSinceLastContact: 260,
	}
	lowStaleConfidence, staleReason := confidenceWithReason(lowSampleStale, 15)

	if lowRecentConfidence >= highRecentConfidence {
		t.Fatalf("low sample confidence %.2f should be lower than high sample %.2f", lowRecentConfidence, highRecentConfidence)
	}
	if lowStaleConfidence >= lowRecentConfidence {
		t.Fatalf("stale confidence %.2f should be lower than recent %.2f", lowStaleConfidence, lowRecentConfidence)
	}
	if !strings.Contains(staleReason, "180") {
		t.Fatalf("expected stale confidence reason to mention heavy freshness decay, got: %s", staleReason)
	}
}

func TestRankScore_CurrentBoardsPenalizeStaleMoreThanCooling(t *testing.T) {
	profile := &relationProfile{
		Recent7Messages:      30,
		Previous30Messages:   64,
		Recent7Sessions:      8,
		Previous30Sessions:   14,
		DaysSinceLastContact: 20,
	}
	currentRecent := relationRankScore("warming", 88, profile)
	historyRecent := relationRankScore("cooling", 88, profile)

	profile.DaysSinceLastContact = 240
	currentStale := relationRankScore("warming", 88, profile)
	historyStale := relationRankScore("cooling", 88, profile)

	if currentStale >= currentRecent {
		t.Fatalf("warming stale rank %.2f should be lower than recent %.2f", currentStale, currentRecent)
	}
	if historyStale >= historyRecent {
		t.Fatalf("cooling stale rank %.2f should still be lower than recent %.2f", historyStale, historyRecent)
	}
	if historyStale <= currentStale {
		t.Fatalf("cooling stale rank %.2f should stay above warming stale rank %.2f for historical tolerance", historyStale, currentStale)
	}
}

func TestCurrentBoardsRequireRecentActivityWhileCoolingCanStay(t *testing.T) {
	profile := &relationProfile{
		TotalMessages:        180,
		TextMessages:         120,
		TotalSessions:        24,
		MyMessages:           90,
		TheirMessages:        90,
		ReplySamples:         make([]float64, 12),
		Recent7Messages:      20,
		Previous30Messages:   50,
		Recent7Sessions:      4,
		Previous30Sessions:   12,
		PeakMonthCount:       72,
		CurrentMonthCount:    18,
		PreviousMonthCount:   30,
		DaysSinceLastContact: 200,
	}

	if shouldIncludeWarmingBoard(profile) {
		t.Fatalf("warming board should reject stale profile")
	}
	if shouldIncludeInitiativeBoard(profile) {
		t.Fatalf("initiative board should reject stale profile")
	}
	if shouldIncludeFastReplyBoard(profile) {
		t.Fatalf("fast-reply board should reject stale profile")
	}
	if !shouldIncludeCoolingBoard(profile) {
		t.Fatalf("cooling board should keep historical profile")
	}
}

func TestBuildStaleHint_HistoricalAndCurrent(t *testing.T) {
	if hint := buildStaleHint(10, false); hint != "" {
		t.Fatalf("expected no hint for fresh contact, got: %s", hint)
	}
	if hint := buildStaleHint(220, true); !strings.Contains(hint, "历史") {
		t.Fatalf("expected historical stale hint, got: %s", hint)
	}
}

func TestFilterProfilesByGender_ExcludeUnknown(t *testing.T) {
	profiles := []*relationProfile{
		{Username: "u_male", Gender: model.GenderMale},
		{Username: "u_female", Gender: model.GenderFemale},
		{Username: "u_unknown", Gender: model.GenderUnknown},
	}

	maleProfiles := filterProfilesByGender(profiles, model.GenderMale)
	if len(maleProfiles) != 1 || maleProfiles[0].Username != "u_male" {
		t.Fatalf("unexpected male profiles: %#v", maleProfiles)
	}

	femaleProfiles := filterProfilesByGender(profiles, model.GenderFemale)
	if len(femaleProfiles) != 1 || femaleProfiles[0].Username != "u_female" {
		t.Fatalf("unexpected female profiles: %#v", femaleProfiles)
	}
}
