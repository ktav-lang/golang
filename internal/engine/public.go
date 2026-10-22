package engine

import "encoding/json"

func loadsPublic(src string, strict bool) (any, error) {
	if err := invalidUTF8Error(src); err != nil {
		return nil, err
	}
	var raw []byte
	var err error
	if strict {
		raw, err = pureLoadsStrictTagged([]byte(src))
	} else {
		raw, err = pureLoadsTagged([]byte(src))
	}
	if err != nil {
		return nil, err
	}
	return decodeJSON(raw)
}
func loadsIntoPublic(src string, target any) error {
	if err := invalidUTF8Error(src); err != nil {
		return err
	}
	raw, err := pureLoadsTagged([]byte(src))
	if err != nil {
		return err
	}
	flat, err := flattenTagged(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(flat, target)
}
func dumpsPublic(v any, force, canonical bool) (string, error) {
	tagged, err := encodeWriterTagged(v, force)
	if err != nil {
		return "", err
	}
	var out []byte
	if canonical {
		out, err = pureEmitCanonicalTagged(tagged)
	} else if force {
		out, err = pureDumpsForceStringsTagged(tagged)
	} else {
		out, err = pureDumpsTagged(tagged)
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}
func formatPublic(src string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	out, err := pureFormatSource([]byte(src))
	return string(out), err
}
func canonicalPublic(src string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	out, err := pureCanonicalFromSource([]byte(src))
	return string(out), err
}
