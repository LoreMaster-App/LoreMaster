package documentconversion

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"

	"lore-master/libs/documentation-sync/platformport"
)

// renderHash is the hash of a converted page body. The encoding names every node's
// type and follows pointers, so "*x*" and "**x**" differ and two runs over the same
// input always agree.
func renderHash(document platformport.Document) string {
	var encoded bytes.Buffer
	encode(&encoded, reflect.ValueOf(document))
	sum := sha256.Sum256(encoded.Bytes())

	return "sha256:" + hex.EncodeToString(sum[:])
}

func encode(out *bytes.Buffer, value reflect.Value) {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			out.WriteString("nil")

			return
		}
		encode(out, value.Elem())
	case reflect.Struct:
		out.WriteString(value.Type().Name())
		out.WriteByte('{')
		for i := range value.NumField() {
			encode(out, value.Field(i))
			out.WriteByte(',')
		}
		out.WriteByte('}')
	case reflect.Slice:
		out.WriteByte('[')
		for i := range value.Len() {
			encode(out, value.Index(i))
			out.WriteByte(',')
		}
		out.WriteByte(']')
	case reflect.String:
		out.WriteString(strconv.Quote(value.String()))
	default:
		fmt.Fprint(out, value)
	}
}
