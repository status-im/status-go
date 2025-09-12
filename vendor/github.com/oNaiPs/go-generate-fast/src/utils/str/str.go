package str

import (
	"fmt"
	"path/filepath"
	"sort"
)

// StringList flattens its arguments into a single []string.
// Each argument in args must have type string or []string.
func StringList(args ...any) []string {
	x := []string{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case []string:
			x = append(x, arg...)
		case string:
			x = append(x, arg)
		default:
			panic("stringList: invalid argument of type " + fmt.Sprintf("%T", arg))
		}
	}
	return x
}

func RemoveDuplicatesAndSort(elements *[]string) {
	seen := make(map[string]struct{})
	result := []string{}

	for v := range *elements {
		if _, ok := seen[(*elements)[v]]; !ok {
			seen[(*elements)[v]] = struct{}{}
			result = append(result, (*elements)[v])
		}
	}

	sort.Strings(result)

	// Update the original slice
	*elements = result
}

func ConvertToRelativePaths(elements *[]string, basepath string) error {
	for i, element := range *elements {
		if !filepath.IsAbs(element) {
			continue // Skip relative paths
		}
		relpath, err := filepath.Rel(basepath, element)
		if err != nil {
			return fmt.Errorf("failed to convert %s to a relative path: %w", element, err)
		}
		(*elements)[i] = relpath
	}
	return nil
}
