package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDisplayName_AcceptsAnyScript(t *testing.T) {
	names := []string{
		"John Doe",
		"john_doe-1",
		"Ђорђе Ћурчић",                            // Serbian Cyrillic
		"Đorđe Čurčić",                            // Serbian Latin
		"Ґудзик Їжак",                             // Ukrainian
		"Straße Ärger",                            // German
		"Élodie Noël",                             // French
		"Αλέξανδρος",                              // Greek
		"我的名字很长",                                  // Chinese
		"財布 ワレット",                                 // Japanese
		"내 이름은 김",                                 // Korean
		"محفظتي هنا",                              // Arabic
		"הארנק שלי",                               // Hebrew
		"मेरा बटुआ",                               // Hindi, with combining marks
		"กระเป๋าของฉัน",                           // Thai, with combining marks
		"Ví của tôi",                              // Vietnamese
		"Ђорђе ٣٤٥",                               // non-ASCII digits
		strings.Repeat("Ж", MaxDisplayNameLength), // 24 runes but 48 bytes
	}
	for _, name := range names {
		n := name
		require.NoError(t, ValidateDisplayName(&n), name)
		require.Equal(t, strings.TrimSpace(name), n)
	}
}

func TestValidateDisplayName_Rejects(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"Ђорђе!", ErrInvalidDisplayNameRegExp},
		{"john.doe", ErrInvalidDisplayNameRegExp},
		{"我的名字。", ErrInvalidDisplayNameRegExp},
		{"Ђорђе 🙂", ErrInvalidDisplayNameRegExp},
		{"Ђорђ", ErrInvalidDisplayNameLength},                                      // 4 runes (8 bytes)
		{strings.Repeat("Ж", MaxDisplayNameLength+1), ErrInvalidDisplayNameLength}, // 25 runes
		{"Ђорђе-eth", ErrInvalidDisplayNameEthSuffix},
	}
	for _, c := range cases {
		n := c.name
		require.ErrorIs(t, ValidateDisplayName(&n), c.err, c.name)
	}
}

func TestValidateDisplayName_EmptyIsAccepted(t *testing.T) {
	n := "   "
	require.NoError(t, ValidateDisplayName(&n))
	require.Equal(t, "", n)
}
