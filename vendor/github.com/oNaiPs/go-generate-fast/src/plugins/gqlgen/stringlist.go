package plugin_gqlgen

type StringList []string

func (a *StringList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	err := unmarshal(&single)
	if err == nil {
		*a = []string{single}
		return nil
	}

	var multi []string
	err = unmarshal(&multi)
	if err != nil {
		return err
	}

	*a = multi
	return nil
}

func (a StringList) Has(file string) bool {
	for _, existing := range a {
		if existing == file {
			return true
		}
	}
	return false
}
