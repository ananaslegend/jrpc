package jrpc

import (
	"fmt"

	"github.com/valyala/fastjson"
)

var (
	nullIDValue = "null"
)

type ID struct {
	stringID *string
	intID    *int
	floatID  *float64
}

func (i *ID) String() string {
	if i.stringID != nil {
		return fmt.Sprintf(`"%s"`, *i.stringID)
	}

	if i.floatID != nil {
		return fmt.Sprintf(`%v`, *i.floatID)
	}

	return fmt.Sprintf(`%v`, *i.intID)
}

func getRequestID(v *fastjson.Value) *ID {
	if !v.Exists("id") {
		return nil
	}

	idValue := v.Get("id")

	intID, err := idValue.Int()
	if err != nil {
		floatID, err := idValue.Float64()
		if err != nil {
			bytesID, err := idValue.StringBytes()
			if err != nil {
				return &ID{stringID: &nullIDValue}
			}

			stringID := string(bytesID)

			return &ID{stringID: &stringID}
		}

		return &ID{floatID: &floatID}
	}

	return &ID{intID: &intID}
}
