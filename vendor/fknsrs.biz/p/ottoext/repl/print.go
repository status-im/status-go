package repl

import (
	"fmt"
	"strings"

	"github.com/robertkrimen/otto"
)

func seenWith(seen map[otto.Value]bool, v otto.Value) map[otto.Value]bool {
	r := make(map[otto.Value]bool)
	for k, v := range seen {
		r[k] = v
	}

	r[v] = true

	return r
}

func format(v otto.Value, width, indent, limit int) (string, error) {
	return formatIndent(v, width, indent, limit, 0, 0, make(map[otto.Value]bool))
}

func formatIndent(v otto.Value, width, indent, limit, level, additional int, seen map[otto.Value]bool) (string, error) {
	if limit == 0 {
		return "...", nil
	}

	switch {
	case v.IsBoolean(), v.IsNull(), v.IsNumber(), v.IsUndefined():
		return v.String(), nil
	case v.IsString():
		return fmt.Sprintf("%q", v.String()), nil
	case v.IsFunction():
		n, err := v.Object().Get("name")
		if err != nil {
			return "", err
		}
		if n.IsUndefined() {
			return "function", nil
		}
		return fmt.Sprintf("function %s", n.String()), nil
	case v.IsObject():
		if d, err := formatOneLine(v, limit, seen); err != nil {
			return "", err
		} else if level*indent+additional+len(d) <= width {
			return d, nil
		}

		switch v.Class() {
		case "Array":
			return formatArray(v, width, indent, limit, level, seen)
		default:
			return formatObject(v, width, indent, limit, level, seen)
		}
	default:
		return "", fmt.Errorf("couldn't format type %s", v.Class())
	}
}

func formatArray(v otto.Value, width, indent, limit, level int, seen map[otto.Value]bool) (string, error) {
	if seen[v] {
		return strings.Repeat(" ", level*indent) + "[circular]", nil
	}

	o := v.Object()

	lv, err := o.Get("length")
	if err != nil {
		return "", err
	}
	li, err := lv.Export()
	if err != nil {
		return "", err
	}
	l, ok := li.(uint32)
	if !ok {
		return "", fmt.Errorf("length property must be a number; was %T", li)
	}

	bits := []string{"["}

	for i := 0; i < int(l); i++ {
		e, err := o.Get(fmt.Sprintf("%d", i))
		if err != nil {
			return "", err
		}

		d, err := formatIndent(e, width, indent, limit-1, level+1, 0, seenWith(seen, v))
		if err != nil {
			return "", err
		}

		bits = append(bits, strings.Repeat(" ", (level+1)*indent)+d+",")
	}

	bits = append(bits, strings.Repeat(" ", level*indent)+"]")

	return strings.Join(bits, "\n"), nil
}

func formatObject(v otto.Value, width, indent, limit, level int, seen map[otto.Value]bool) (string, error) {
	if seen[v] {
		return strings.Repeat(" ", level*indent) + "[circular]", nil
	}

	o := v.Object()

	bits := []string{"{"}

	keys := o.Keys()

	for i, k := range keys {
		e, err := o.Get(k)

		d, err := formatIndent(e, width, indent, limit-1, level+1, len(k)+2, seenWith(seen, v))
		if err != nil {
			return "", err
		}

		bits = append(bits, strings.Repeat(" ", (level+1)*indent)+k+": "+d+",")

		i++
	}

	bits = append(bits, strings.Repeat(" ", level*indent)+"}")

	return strings.Join(bits, "\n"), nil
}

func formatOneLine(v otto.Value, limit int, seen map[otto.Value]bool) (string, error) {
	if limit == 0 {
		return "...", nil
	}

	switch {
	case v.IsBoolean(), v.IsNull(), v.IsNumber(), v.IsUndefined():
		return v.String(), nil
	case v.IsString():
		return fmt.Sprintf("%q", v.String()), nil
	case v.IsFunction():
		n, err := v.Object().Get("name")
		if err != nil {
			return "", err
		}
		if n.IsUndefined() {
			return "function", nil
		}
		return fmt.Sprintf("function %s", n.String()), nil
	case v.IsObject():
		switch v.Class() {
		case "Array":
			return formatArrayOneLine(v, limit, seen)
		default:
			return formatObjectOneLine(v, limit, seen)
		}
	default:
		return "", fmt.Errorf("couldn't format type %s", v.Class())
	}
}

func formatArrayOneLine(v otto.Value, limit int, seen map[otto.Value]bool) (string, error) {
	if limit == 0 {
		return "...", nil
	}

	if seen[v] {
		return "[circular]", nil
	}

	o := v.Object()

	lv, err := o.Get("length")
	if err != nil {
		return "", err
	}
	li, err := lv.Export()
	if err != nil {
		return "", err
	}
	l, ok := li.(uint32)
	if !ok {
		return "", fmt.Errorf("length property must be a number; was %T", li)
	}

	var bits []string

	for i := 0; i < int(l); i++ {
		e, err := o.Get(fmt.Sprintf("%d", i))
		if err != nil {
			return "", err
		}

		d, err := formatOneLine(e, limit-1, seenWith(seen, v))
		if err != nil {
			return "", err
		}

		bits = append(bits, d)
	}

	return "[" + strings.Join(bits, ", ") + "]", nil
}

func formatObjectOneLine(v otto.Value, limit int, seen map[otto.Value]bool) (string, error) {
	if limit == 0 {
		return "...", nil
	}

	if seen[v] {
		return "[circular]", nil
	}

	o := v.Object()

	bits := []string{}

	keys := o.Keys()

	for _, k := range keys {
		e, err := o.Get(k)
		if err != nil {
			return "", err
		}

		d, err := formatOneLine(e, limit-1, seenWith(seen, v))
		if err != nil {
			return "", err
		}

		bits = append(bits, k+": "+d)
	}

	return "{" + strings.Join(bits, ", ") + "}", nil
}
