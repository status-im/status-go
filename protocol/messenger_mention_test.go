package protocol

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
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
			Contact: &Contact{
				ID:            "0xpk1",
				LocalNickname: "User Number One",
			},
		},
		"0xpk2": {
			Contact: &Contact{
				ID:            "0xpk2",
				LocalNickname: "user2",
				ENSVerified:   true,
				EnsName:       "User Number Two",
			},
		},
		"0xpk3": {
			Contact: &Contact{
				ID:            "0xpk3",
				LocalNickname: "user3",
				ENSVerified:   true,
				EnsName:       "User Number Three",
			},
		},
		"0xpk4": {
			Contact: &Contact{
				ID:            "0xpk4",
				EnsName:       "ens-user-4.eth",
				ENSVerified:   true,
				DisplayName:   "display-name-user-4",
				LocalNickname: "primary-name-user-4",
			},
		},
		"0xpk5": {
			Contact: &Contact{
				ID:            "0xpk5",
				LocalNickname: "User Number",
			},
		},
		"0xpk6": {
			Contact: &Contact{
				ID:            "0xpk6",
				LocalNickname: "特别字符",
				DisplayName:   "特别 字符",
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
		{"starts with mention but no space after", "@User NumberOnefoo", "@User NumberOnefoo"},
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

		{"double @", "@ @user2", "@ @0xpk2"},

		{"user name contains dash", "@display-name-user-4 ", "@0xpk4 "},

		{"username or nickname of one is a substring of another case 1", "@User Number One @User Number", "@0xpk1 @0xpk5"},
		{"username or nickname of one is a substring of another case 2", "@User Number @User Number One ", "@0xpk5 @0xpk1 "},

		{"special chars in username case1", "@特别字符", "@0xpk6"},
		{"special chars in username case2", "@特别字符 ", "@0xpk6 "},
		{"special chars in username case3", " @特别 字符 ", " @0xpk6 "},
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
			NewText:      newText,
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
	testCases := []struct {
		name     string
		input    string
		expected []InputSegment
	}{
		{
			"only text",
			"parse-text",
			[]InputSegment{{Type: Text, Value: "parse-text"}},
		},
		{
			"in the middle",
			"hey @0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073 he",
			[]InputSegment{
				{Type: Text, Value: "hey "},
				{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
				{Type: Text, Value: " he"},
			},
		},
		{
			"at the beginning",
			"@0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073 he",
			[]InputSegment{
				{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
				{Type: Text, Value: " he"},
			},
		},
		{
			"at the end",
			"hey @0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073",
			[]InputSegment{
				{Type: Text, Value: "hey "},
				{Type: Mention, Value: "0x04fbce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
			},
		},
		{
			"invalid",
			"invalid @0x04fBce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073",
			[]InputSegment{
				{Type: Text, Value: "invalid @0x04fBce10971e1cd7253b98c7b7e54de3729ca57ce41a2bfb0d1c4e0a26f72c4b6913c3487fa1b4bb86125770f1743fb4459da05c1cbe31d938814cfaf36e252073"},
			},
		},
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

func TestSubs(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Normal case",
			input:    "Hello, world!",
			start:    0,
			end:      5,
			expected: "Hello",
		},
		{
			name:     "Start index out of range (negative)",
			input:    "Hello, world!",
			start:    -5,
			end:      5,
			expected: "Hello",
		},
		{
			name:     "End index out of range",
			input:    "Hello, world!",
			start:    7,
			end:      50,
			expected: "world!",
		},
		{
			name:     "Start index greater than end index",
			input:    "Hello, world!",
			start:    10,
			end:      5,
			expected: ", wor",
		},
		{
			name:     "Both indices out of range",
			input:    "Hello, world!",
			start:    -5,
			end:      50,
			expected: "Hello, world!",
		},
		{
			name:     "Start index negative, end index out of range",
			input:    "Hello, world!",
			start:    -10,
			end:      15,
			expected: "Hello, world!",
		},
		{
			name:     "Start index negative, end index within range",
			input:    "Hello, world!",
			start:    -10,
			end:      5,
			expected: "Hello",
		},
		{
			name:     "Start index negative, end index negative",
			input:    "Hello, world!",
			start:    -10,
			end:      -5,
			expected: "",
		},

		{
			name:     "Start index zero, end index zero",
			input:    "Hello, world!",
			start:    0,
			end:      0,
			expected: "",
		},
		{
			name:     "Start index positive, end index zero",
			input:    "Hello, world!",
			start:    3,
			end:      0,
			expected: "Hel",
		},
		{
			name:     "Start index equal to input length",
			input:    "Hello, world!",
			start:    13,
			end:      15,
			expected: "",
		},
		{
			name:     "End index negative",
			input:    "Hello, world!",
			start:    5,
			end:      -5,
			expected: "Hello",
		},
		{
			name:     "Start and end indices equal and negative",
			input:    "Hello, world!",
			start:    -3,
			end:      -3,
			expected: "",
		},
		{
			name:     "Start index greater than input length",
			input:    "Hello, world!",
			start:    15,
			end:      20,
			expected: "",
		},
		{
			name:     "End index equal to input length",
			input:    "Hello, world!",
			start:    0,
			end:      13,
			expected: "Hello, world!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := subs(tc.input, tc.start, tc.end)
			if actual != tc.expected {
				t.Errorf("Test case '%s': expected '%s', got '%s'", tc.name, tc.expected, actual)
			}
		})
	}
}

func TestLastIndexOf(t *testing.T) {
	atSignIdx := lastIndexOfAtSign("@", 0)
	require.Equal(t, 0, atSignIdx)

	atSignIdx = lastIndexOfAtSign("@@", 1)
	require.Equal(t, 1, atSignIdx)

	//at-sign-idx 0 text @t searched-text t start 2 end 2 new-text
	atSignIdx = lastIndexOfAtSign("@t", 2)
	require.Equal(t, 0, atSignIdx)

	atSignIdx = lastIndexOfAtSign("at", 3)
	require.Equal(t, -1, atSignIdx)
}

func TestDiffText(t *testing.T) {
	testCases := []struct {
		oldText  string
		newText  string
		expected *TextDiff
	}{
		{
			oldText: "",
			newText: "A",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "",
				newText:      "A",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "A",
			newText: "Ab",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "A",
				newText:      "b",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Ab",
			newText: "Abc",
			expected: &TextDiff{
				start:        2,
				end:          2,
				previousText: "Ab",
				newText:      "c",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Ab",
			newText: "cAb",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "Ab",
				newText:      "c",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Ac",
			newText: "Adc",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "Ac",
				newText:      "d",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Adc",
			newText: "Ad ee c",
			expected: &TextDiff{
				start:        2,
				end:          2,
				previousText: "Adc",
				newText:      " ee ",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Ad ee c",
			newText: "A fff d ee c",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "Ad ee c",
				newText:      " fff ",
				operation:    textOperationAdd,
			},
		},
		{
			oldText: "Abc",
			newText: "Ac",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "Abc",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: "Abcd",
			newText: "Ab",
			expected: &TextDiff{
				start:        2,
				end:          3,
				previousText: "Abcd",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: "Abcd",
			newText: "bcd",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "Abcd",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: "Abcd你好",
			newText: "Abcd你",
			expected: &TextDiff{
				start:        5,
				end:          5,
				previousText: "Abcd你好",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: "Abcd你好",
			newText: "Abcd",
			expected: &TextDiff{
				start:        4,
				end:          5,
				previousText: "Abcd你好",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: "A fff d ee c",
			newText: " fff d ee c",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "A fff d ee c",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText: " fff d ee c",
			newText: " fffee c",
			expected: &TextDiff{
				start:        4,
				end:          6,
				previousText: " fff d ee c",
				newText:      "",
				operation:    textOperationDelete,
			},
		},
		{
			oldText:  "abc",
			newText:  "abc",
			expected: nil,
		},
		{
			oldText: "abc",
			newText: "ghij",
			expected: &TextDiff{
				start:        0,
				end:          2,
				previousText: "abc",
				newText:      "ghij",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "babcd",
			expected: &TextDiff{
				start:        0,
				end:          2,
				previousText: "abc",
				newText:      "babcd",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "baebcd",
			expected: &TextDiff{
				start:        0,
				end:          2,
				previousText: "abc",
				newText:      "baebcd",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "aefc",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "abc",
				newText:      "ef",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "adc",
			expected: &TextDiff{
				start:        1,
				end:          1,
				previousText: "abc",
				newText:      "d",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "abd",
			expected: &TextDiff{
				start:        2,
				end:          2,
				previousText: "abc",
				newText:      "d",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "cbc",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "abc",
				newText:      "c",
				operation:    textOperationReplace,
			},
		},
		{
			oldText: "abc",
			newText: "ffbc",
			expected: &TextDiff{
				start:        0,
				end:          0,
				previousText: "abc",
				newText:      "ff",
				operation:    textOperationReplace,
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			diff := diffText(tc.oldText, tc.newText)
			require.Equal(t, tc.expected, diff)
		})
	}
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

func TestMentionSuggestionCases(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest(t, nil)

	testCases := []struct {
		inputText    string
		expectedSize int
	}{
		{"@", len(mentionableUserMap)},
		{"@u", len(mentionableUserMap)},
		{"@u2", 1},
		{"@u23", 0},
		{"@u2", 1},
		{"@u2 abc", 0},
		{"@u2 abc @u3", 1},
		{"@u2 abc@u3", 0},
		{"@u2 abc@u3 ", 0},
		{"@u2 abc @u3", 1},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			ctx, err := mentionManager.OnChangeText(chatID, tc.inputText)
			require.NoError(t, err)
			t.Logf("Input: %+v, MentionState:%+v, InputSegments:%+v\n", tc.inputText, ctx.MentionState, ctx.InputSegments)
			require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
		})
	}
}

func TestMentionSuggestionAfterToInputField(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest(t, nil)
	_, err := mentionManager.ToInputField(chatID, "abc")
	require.NoError(t, err)
	ctx, err := mentionManager.OnChangeText(chatID, "@")
	require.NoError(t, err)
	require.Equal(t, len(mentionableUserMap), len(ctx.MentionSuggestions))
}

func TestMentionSuggestionSpecialInputModeForAndroid(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest(t, nil)

	testCases := []struct {
		inputText    string
		expectedSize int
	}{
		{"A", 0},
		{"As", 0},
		{"Asd", 0},
		{"Asd@", len(mentionableUserMap)},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			ctx, err := mentionManager.OnChangeText(chatID, tc.inputText)
			require.NoError(t, err)
			require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
			t.Logf("Input: %+v, MentionState:%+v, InputSegments:%+v\n", tc.inputText, ctx.MentionState, ctx.InputSegments)
		})
	}
}

func TestMentionSuggestionSpecialChars(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest(t, nil)

	testCases := []struct {
		inputText    string
		expectedSize int
	}{
		{"'", 0},
		{"‘", 0},
		{"‘@", len(mentionableUserMap)},
		{"‘@自由人", 1},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			ctx, err := mentionManager.OnChangeText(chatID, tc.inputText)
			require.NoError(t, err)
			t.Logf("Input: %+v, MentionState:%+v, InputSegments:%+v\n", tc.inputText, ctx.MentionState, ctx.InputSegments)
			require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
		})
	}
}

func TestMentionSuggestionAtSignSpaceCases(t *testing.T) {
	mentionableUserMap, chatID, mentionManager := setupMentionSuggestionTest(t, map[string]*MentionableUser{
		"0xpk1": {
			Contact: &Contact{
				ID:            "0xpk1",
				LocalNickname: "User Number One",
			},
		},
	})

	testCases := []struct {
		inputText    string
		expectedSize int
	}{
		{"@", len(mentionableUserMap)},
		{"@ ", 0},
		{"@ @", len(mentionableUserMap)},
	}

	var ctx *ChatMentionContext
	var err error
	for _, tc := range testCases {
		ctx, err = mentionManager.OnChangeText(chatID, tc.inputText)
		require.NoError(t, err)
		t.Logf("After OnChangeText, Input: %+v, MentionState:%+v, InputSegments:%+v\n", tc.inputText, ctx.MentionState, ctx.InputSegments)
		require.Equal(t, tc.expectedSize, len(ctx.MentionSuggestions))
	}
	require.Len(t, ctx.InputSegments, 2)
	require.Equal(t, Text, ctx.InputSegments[0].Type)
	require.Equal(t, "@ ", ctx.InputSegments[0].Value)
	require.Equal(t, Text, ctx.InputSegments[1].Type)
	require.Equal(t, "@", ctx.InputSegments[1].Value)
}

func TestSelectMention(t *testing.T) {
	mentionableUsers, chatID, mentionManager := setupMentionSuggestionTest(t, nil)

	text := "@u2 abc"
	ctx, err := mentionManager.OnChangeText(chatID, text)
	require.NoError(t, err)
	require.Equal(t, 0, len(ctx.MentionSuggestions))

	ctx, err = mentionManager.OnChangeText(chatID, "@u abc")
	require.NoError(t, err)
	require.Equal(t, len(mentionableUsers), len(ctx.MentionSuggestions))

	ctx, err = mentionManager.SelectMention(chatID, "@u abc", "u2", "0xpk2")
	require.NoError(t, err)
	require.Equal(t, 0, len(ctx.MentionSuggestions))
	require.Equal(t, text, ctx.NewText)
	require.Equal(t, text, ctx.PreviousText)

	ctx, err = mentionManager.OnChangeText(chatID, text)
	require.NoError(t, err)
	require.Equal(t, 0, len(ctx.MentionSuggestions))
}

func TestInputSegments(t *testing.T) {
	_, chatID, mentionManager := setupMentionSuggestionTest(t, nil)
	ctx, err := mentionManager.OnChangeText(chatID, "@u1")
	require.NoError(t, err)
	require.Equal(t, 1, len(ctx.InputSegments))
	require.Equal(t, Text, ctx.InputSegments[0].Type)
	require.Equal(t, "@u1", ctx.InputSegments[0].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u1 @User Number One")
	require.NoError(t, err)
	require.Equal(t, 2, len(ctx.InputSegments))
	require.Equal(t, Text, ctx.InputSegments[0].Type)
	require.Equal(t, "@u1 ", ctx.InputSegments[0].Value)
	require.Equal(t, Mention, ctx.InputSegments[1].Type)
	require.Equal(t, "@User Number One", ctx.InputSegments[1].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u1 @User Number O")
	require.NoError(t, err)
	require.Equal(t, 2, len(ctx.InputSegments))
	require.Equal(t, Text, ctx.InputSegments[1].Type)
	require.Equal(t, "@User Number O", ctx.InputSegments[1].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u2 @User Number One")
	require.NoError(t, err)
	require.Equal(t, 3, len(ctx.InputSegments))
	require.Equal(t, Mention, ctx.InputSegments[0].Type)
	require.Equal(t, "@u2", ctx.InputSegments[0].Value)
	require.Equal(t, Text, ctx.InputSegments[1].Type)
	require.Equal(t, " ", ctx.InputSegments[1].Value)
	require.Equal(t, Mention, ctx.InputSegments[2].Type)
	require.Equal(t, "@User Number One", ctx.InputSegments[2].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u2 @User Number One a ")
	require.NoError(t, err)
	require.Equal(t, 4, len(ctx.InputSegments))
	require.Equal(t, Mention, ctx.InputSegments[2].Type)
	require.Equal(t, "@User Number One", ctx.InputSegments[2].Value)
	require.Equal(t, Text, ctx.InputSegments[3].Type)
	require.Equal(t, " a ", ctx.InputSegments[3].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u2 @User Numbed One a ")
	require.NoError(t, err)
	require.Equal(t, 3, len(ctx.InputSegments))
	require.Equal(t, Mention, ctx.InputSegments[0].Type)
	require.Equal(t, "@u2", ctx.InputSegments[0].Value)
	require.Equal(t, Text, ctx.InputSegments[2].Type)
	require.Equal(t, "@User Numbed One a ", ctx.InputSegments[2].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@ @ ")
	require.NoError(t, err)
	require.Equal(t, 2, len(ctx.InputSegments))
	require.Equal(t, Text, ctx.InputSegments[0].Type)
	require.Equal(t, "@ ", ctx.InputSegments[0].Value)
	require.Equal(t, Text, ctx.InputSegments[1].Type)
	require.Equal(t, "@ ", ctx.InputSegments[1].Value)

	ctx, err = mentionManager.OnChangeText(chatID, "@u3 @ ")
	require.NoError(t, err)
	require.Equal(t, 3, len(ctx.InputSegments))
	require.Equal(t, Mention, ctx.InputSegments[0].Type)
	require.Equal(t, "@u3", ctx.InputSegments[0].Value)
	require.Equal(t, Text, ctx.InputSegments[1].Type)
	require.Equal(t, " ", ctx.InputSegments[1].Value)
	require.Equal(t, Text, ctx.InputSegments[2].Type)
	require.Equal(t, "@ ", ctx.InputSegments[2].Value)
}

func setupMentionSuggestionTest(t *testing.T, mentionableUserMapInput map[string]*MentionableUser) (map[string]*MentionableUser, string, *MentionManager) {
	mentionableUserMap := mentionableUserMapInput
	if mentionableUserMap == nil {
		mentionableUserMap = getDefaultMentionableUserMap()
	}

	for _, u := range mentionableUserMap {
		addSearchablePhrases(u)
	}

	mockMentionableUserGetter := &MockMentionableUserGetter{
		mentionableUserMap: mentionableUserMap,
	}

	chatID := "0xchatID"
	allChats := new(chatMap)
	allChats.Store(chatID, &Chat{})

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	mentionManager := &MentionManager{
		mentionableUserGetter: mockMentionableUserGetter,
		mentionContexts:       make(map[string]*ChatMentionContext),
		Messenger: &Messenger{
			allChats: allChats,
			identity: key,
		},
		logger: logutils.ZapLogger().Named("MentionManager"),
	}

	return mentionableUserMap, chatID, mentionManager
}

func getDefaultMentionableUserMap() map[string]*MentionableUser {
	return map[string]*MentionableUser{
		"0xpk1": {
			Contact: &Contact{
				ID:            "0xpk1",
				LocalNickname: "User Number One",
			},
		},
		"0xpk2": {
			Contact: &Contact{
				ID:            "0xpk2",
				LocalNickname: "u2",
				ENSVerified:   true,
				EnsName:       "User Number Two",
			},
		},
		"0xpk3": {
			Contact: &Contact{
				ID:            "0xpk3",
				LocalNickname: "u3",
				ENSVerified:   true,
				EnsName:       "User Number Three",
			},
		},
		"0xpk4": {
			Contact: &Contact{
				ID:            "0xpk4",
				LocalNickname: "自由人",
				ENSVerified:   true,
				EnsName:       "User Number Four",
			},
		},
	}
}
