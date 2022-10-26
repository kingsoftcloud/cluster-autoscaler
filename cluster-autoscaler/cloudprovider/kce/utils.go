package kce

import (
	"bytes"
	"encoding/gob"
	"strings"
)
// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

func ParseNodeGroupId(id string) (string, string) {
	if strings.Contains(id, NodeGroupIdZoneSeparator) {
		parts := strings.Split(id, NodeGroupIdZoneSeparator)
		return parts[0], parts[1]
	} else {
		return id, ""
	}
}

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}













