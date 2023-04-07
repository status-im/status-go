package protocol

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRePosRegex(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"@", true},
		{"~", true},
		{"\\", true},
		{"*", true},
		{"_", true},
		{"\n", true},
		{"`", true},
		{"a", false},
		{"#", false},
	}

	for _, tc := range testCases {
		actual := specialCharsRegex.MatchString(tc.input)
		if actual != tc.expected {
			t.Errorf("Unexpected match result for input '%s': expected=%v, actual=%v", tc.input, tc.expected, actual)
		}
	}
}

func TestRePos(t *testing.T) {
	// Test case 1: Empty string
	s1 := ""
	var want1 []specialCharLocation
	if got1 := rePos(s1); !reflect.DeepEqual(got1, want1) {
		t.Errorf("rePos(%q) = %v, want %v", s1, got1, want1)
	}

	// Test case 2: Single match
	s2 := "test@string"
	want2 := []specialCharLocation{{4, "@"}}
	if got2 := rePos(s2); !reflect.DeepEqual(got2, want2) {
		t.Errorf("rePos(%q) = %v, want %v", s2, got2, want2)
	}

	// Test case 3: Multiple matches
	s3 := "this is a test string@with multiple@matches"
	want3 := []specialCharLocation{{21, "@"}, {35, "@"}}
	if got3 := rePos(s3); !reflect.DeepEqual(got3, want3) {
		t.Errorf("rePos(%q) = %v, want %v", s3, got3, want3)
	}

	// Test case 4: No matches
	s4 := "this is a test string with no matches"
	var want4 []specialCharLocation
	if got4 := rePos(s4); !reflect.DeepEqual(got4, want4) {
		t.Errorf("rePos(%q) = %v, want %v", s4, got4, want4)
	}

	// Test case 5: Matches at the beginning and end
	s5 := "@this is a test string@"
	want5 := []specialCharLocation{{0, "@"}, {22, "@"}}
	if got5 := rePos(s5); !reflect.DeepEqual(got5, want5) {
		t.Errorf("rePos(%q) = %v, want %v", s5, got5, want5)
	}

	// Test case 6: special characters
	s6 := "Привет @testm1 "
	want6 := []specialCharLocation{{7, "@"}}
	if got6 := rePos(s6); !reflect.DeepEqual(got6, want6) {
		t.Errorf("rePos(%q) = %v, want %v", s6, got6, want6)
	}

}

func TestReplaceMentions(t *testing.T) {
	users := map[string]*MentionableUser{
		"0xpk1": {
			primaryName: "User Number One",
			Contact: &Contact{
				ID: "0xpk1",
			},
		},
		"0xpk2": {
			primaryName:   "user2",
			secondaryName: "User Number Two",
			Contact: &Contact{
				ID: "0xpk2",
			},
		},
		"0xpk3": {
			primaryName:   "user3",
			secondaryName: "User Number Three",
			Contact: &Contact{
				ID: "0xpk3",
			},
		},
	}

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"empty string", "", ""},
		{"no text", "", ""},
		{"incomlepte mention 1", "@", "@"},
		{"incomplete mention 2", "@r", "@r"},
		{"no mentions", "foo bar @buzz kek @foo", "foo bar @buzz kek @foo"},
		{"starts with mention", "@User Number One", "@0xpk1"},
		{"starts with mention, comma after mention", "@User Number One,", "@0xpk1,"},
		{"starts with mention but no space after", "@User Number Onefoo", "@User Number Onefoo"},
		{"starts with mention, some text after mention", "@User Number One foo", "@0xpk1 foo"},
		{"starts with some text, then mention", "text @User Number One", "text @0xpk1"},
		{"starts with some text, then mention, then more text", "text @User Number One foo", "text @0xpk1 foo"},
		{"no space before mention", "text@User Number One", "text@0xpk1"},
		{"two different mentions", "@User Number One @User Number two", "@0xpk1 @0xpk2"},
		{"two different mentions, separated with comma", "@User Number One,@User Number two", "@0xpk1,@0xpk2"},
		{"two different mentions inside text", "foo@User Number One bar @User Number two baz", "foo@0xpk1 bar @0xpk2 baz"},
		{"ens mention", "@user2", "@0xpk2"},
		{"multiple mentions", strings.Repeat("@User Number One @User Number two ", 1000), strings.Repeat("@0xpk1 @0xpk2 ", 1000)},

		{"single * case 1", "*@user2*", "*@user2*"},
		{"single * case 2", "*@user2 *", "*@0xpk2 *"},
		{"single * case 3", "a*@user2*", "a*@user2*"},
		{"single * case 4", "*@user2 foo*foo", "*@0xpk2 foo*foo"},
		{"single * case 5", "a *@user2*", "a *@user2*"},
		{"single * case 6", "*@user2 foo*", "*@user2 foo*"},
		{"single * case 7", "@user2 *@user2 foo* @user2", "@0xpk2 *@user2 foo* @0xpk2"},
		{"single * case 8", "*@user2 foo**@user2 foo*", "*@user2 foo**@user2 foo*"},
		{"single * case 9", "*@user2 foo***@user2 foo* @user2", "*@user2 foo***@user2 foo* @0xpk2"},

		{"double * case 1", "**@user2**", "**@user2**"},
		{"double * case 2", "**@user2 **", "**@0xpk2 **"},
		{"double * case 3", "a**@user2**", "a**@user2**"},
		{"double * case 4", "**@user2 foo**foo", "**@user2 foo**foo"},
		{"double * case 5", "a **@user2**", "a **@user2**"},
		{"double * case 6", "**@user2 foo**", "**@user2 foo**"},
		{"double * case 7", "@user2 **@user2 foo** @user2", "@0xpk2 **@user2 foo** @0xpk2"},
		{"double * case 8", "**@user2 foo****@user2 foo**", "**@user2 foo****@user2 foo**"},
		{"double * case 9", "**@user2 foo*****@user2 foo** @user2", "**@user2 foo*****@user2 foo** @0xpk2"},

		{"tripple * case 1", "***@user2 foo***@user2 foo*", "***@user2 foo***@0xpk2 foo*"},
		{"tripple ~ case 1", "~~~@user2 foo~~~@user2 foo~", "~~~@user2 foo~~~@user2 foo~"},

		{"quote case 1", ">@user2", ">@user2"},
		{"quote case 2", "\n>@user2", "\n>@user2"},
		{"quote case 3", "\n> @user2 \n   \n @user2", "\n> @user2 \n   \n @0xpk2"},
		{"quote case 4", ">@user2\n\n>@user2", ">@user2\n\n>@user2"},
		{"quote case 5", "***hey\n\n>@user2\n\n@user2 foo***", "***hey\n\n>@user2\n\n@0xpk2 foo***"},

		{"code case 1", "` @user2 `", "` @user2 `"},
		{"code case 2", "` @user2 `", "` @user2 `"},
		{"code case 3", "``` @user2 ```", "``` @user2 ```"},
		{"code case 4", "` ` @user2 ``", "` ` @0xpk2 ``"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceMentions(tt.text, users)
			if got != tt.expected {
				t.Errorf("testing %q, ReplaceMentions(%q) got %q, expected %q", tt.name, tt.text, got, tt.expected)
			}
		})
	}
}

func TestGetAtSignIdxs(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		start int
		want  []int
	}{
		{
			name:  "no @ sign",
			text:  "hello world",
			start: 0,
			want:  []int{},
		},
		{
			name:  "single @ sign",
			text:  "hello @world",
			start: 0,
			want:  []int{6},
		},
		{
			name:  "multiple @ signs",
			text:  "@hello @world @again",
			start: 0,
			want:  []int{0, 7, 14},
		},
		{
			name:  "start after first @ sign",
			text:  "hello @world",
			start: 6,
			want:  []int{12},
		},
		{
			name:  "start after second @ sign",
			text:  "hello @world @again",
			start: 8,
			want:  []int{14, 21},
		},
		{
			name:  "start after last @ sign",
			text:  "hello @world @again",
			start: 15,
			want:  []int{21, 28},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAtSignIdxs(tt.text, tt.start)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAtSignIdxs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalcAtIdxs(t *testing.T) {
	newText := "@abc"
	state := MentionState{
		AtIdxs: []*AtIndexEntry{
			{From: 0, To: 3, Checked: false},
		},
		NewText:      &newText,
		PreviousText: "",
		Start:        0,
	}
	want := []*AtIndexEntry{
		{From: 0, To: 0, Checked: false},
		{From: 4, To: 7, Checked: false},
	}

	got := calcAtIdxs(&state)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("calcAtIdxs() = %v, want %v", got, want)
	}
}

func TestToInfo(t *testing.T) {
	newText := " "
	t.Run("toInfo base case", func(t *testing.T) {
		expected := &MentionState{
			AtSignIdx: 2,
			AtIdxs: []*AtIndexEntry{
				{
					Checked:   true,
					Mentioned: true,
					From:      2,
					To:        17,
				},
			},
			MentionEnd:   19,
			PreviousText: "",
			NewText:      &newText,
			Start:        18,
			End:          18,
		}

		inputSegments := []InputSegment{
			{Type: Text, Value: "H."},
			{Type: Mention, Value: "@helpinghand.eth"},
			{Type: Text, Value: " "},
		}

		actual := toInfo(inputSegments)

		if !reflect.DeepEqual(expected.AtIdxs, actual.AtIdxs) {
			t.Errorf("Expected AtIdxs: %#v, but got: %#v", expected.AtIdxs, actual.AtIdxs)
		}

		expected.AtIdxs = nil
		actual.AtIdxs = nil

		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Expected %#v, but got %#v", expected, actual)
		}
	})
}

func TestToInputField(t *testing.T) {
	mentionText1 := "parse-text"
	mentionTextResult1 := []InputSegment{
		{Type: Text, Value: "parse-text"},
	}

	mentionText2 := "hey @0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073 he"
	mentionTextResult2 := []InputSegment{
		{Type: Text, Value: "hey "},
		{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
		{Type: Text, Value: " he"},
	}

	mentionText3 := "@0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073 he"
	mentionTextResult3 := []InputSegment{
		{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
		{Type: Text, Value: " he"},
	}

	mentionText4 := "hey @0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"
	mentionTextResult4 := []InputSegment{
		{Type: Text, Value: "hey "},
		{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
	}

	mentionText5 := "invalid @0x04fBce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"
	mentionTextResult5 := []InputSegment{
		{Type: Text, Value: "invalid @0x04fBce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
	}

	testCases := []struct {
		name     string
		input    string
		expected []InputSegment
	}{
		{"only text", mentionText1, mentionTextResult1},
		{"in the middle", mentionText2, mentionTextResult2},
		{"at the beginning", mentionText3, mentionTextResult3},
		{"at the end", mentionText4, mentionTextResult4},
		{"invalid", mentionText5, mentionTextResult5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toInputField(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected: %v, got: %v", tc.expected, result)
			}
		})
	}
}

func TestLastIndexOf(t *testing.T) {
	atSignIdx := LastIndexOf("@", charAtSign, 0)
	require.Equal(t, 0, atSignIdx)

	atSignIdx = LastIndexOf("@@", charAtSign, 1)
	require.Equal(t, 1, atSignIdx)

	//at-sign-idx 0 text @t searched-text t start 2 end 2 new-text
	atSignIdx = LastIndexOf("@t", charAtSign, 2)
	require.Equal(t, 0, atSignIdx)
}

type MockMentionableUserGetter struct {
	mentionableUserMap map[string]*MentionableUser
}

func (m *MockMentionableUserGetter) getMentionableUsers(chatID string) (map[string]*MentionableUser, error) {
	return m.mentionableUserMap, nil
}

func (m *MockMentionableUserGetter) getMentionableUser(chatID string, pk string) (*MentionableUser, error) {
	return m.mentionableUserMap[pk], nil
}

func TestMentionSuggestion(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest()

	testCases := []struct {
		newText      string
		prevText     string
		start        int
		end          int
		inputText    string
		expectedSize int
	}{
		{"@", "", 0, 0, "@", len(mentionableUserMap)},
		{"u", "", 1, 1, "@u", len(mentionableUserMap)},
		{"2", "", 2, 2, "@u2", 1},
		{"3", "", 3, 3, "@u23", 0},
		{"", "3", 3, 4, "@u2", 1},
	}

	for _, tc := range testCases {
		input := MentionState{PreviousText: tc.prevText, NewText: &tc.newText, Start: tc.start, End: tc.end}
		_, err := mentionManager.OnTextInput(chatID, &input)
		require.NoError(t, err)
		ctx, err := mentionManager.CalculateSuggestions(chatID, tc.inputText)
		require.NoError(t, err)
		require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
		t.Logf("Input: %+v, MentionState:%+v, InputSegments:%+v\n", input, ctx.MentionState, ctx.InputSegments)
	}
}

func TestMentionSuggestionWithSpecialCharacters(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest()

	testCases := []struct {
		newText             string
		prevText            string
		start               int
		end                 int
		inputText           string
		expectedSize        int
		calculateSuggestion bool
	}{
		{"'", "", 0, 0, "'", 0, false},
		{"‘", "", 0, 1, "‘", 0, true},
		{"@", "", 1, 1, "‘@", len(mentionableUserMap), true},
	}

	for _, tc := range testCases {
		input := MentionState{PreviousText: tc.prevText, NewText: &tc.newText, Start: tc.start, End: tc.end}
		ctx, err := mentionManager.OnTextInput(chatID, &input)
		require.NoError(t, err)
		if tc.calculateSuggestion {
			ctx, err = mentionManager.CalculateSuggestions(chatID, tc.inputText)
			require.NoError(t, err)
			require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
		}
		t.Logf("Input: %+v, MentionState:%+v, InputSegments:%+v\n", input, ctx.MentionState, ctx.InputSegments)
	}
}

func setupMentionSuggestionTest() (map[string]*MentionableUser, string, *MentionManager) {
	mentionableUserMap := getMentionableUserMap()

	for _, u := range mentionableUserMap {
		addSearchablePhrases(u)
	}

	mockMentionableUserGetter := &MockMentionableUserGetter{
		mentionableUserMap: mentionableUserMap,
	}

	chatID := "0xchatID"
	allChats := new(chatMap)
	allChats.Store(chatID, &Chat{})

	mentionManager := &MentionManager{
		mentionableUserGetter: mockMentionableUserGetter,
		mentionContexts:       make(map[string]*ChatMentionContext),
		Messenger: &Messenger{
			allChats: allChats,
		},
	}

	return mentionableUserMap, chatID, mentionManager
}

func getMentionableUserMap() map[string]*MentionableUser {
	return map[string]*MentionableUser{
		"0xpk1": {
			primaryName: "User Number One",
			Contact: &Contact{
				ID: "0xpk1",
			},
		},
		"0xpk2": {
			primaryName:   "u2",
			secondaryName: "User Number Two",
			Contact: &Contact{
				ID: "0xpk2",
			},
		},
		"0xpk3": {
			primaryName:   "u3",
			secondaryName: "User Number Three",
			Contact: &Contact{
				ID: "0xpk3",
			},
		},
	}
}
