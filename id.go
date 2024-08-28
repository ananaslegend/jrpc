package jrpc

import (
	"context"
	"fmt"

	"github.com/valyala/fastjson"
)

const (
	NullRequestID = "null"
)

type idKey struct{}

func setRequestID(ctx context.Context, id *requestID) context.Context {
	return context.WithValue(ctx, idKey{}, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(idKey{}).(*requestID)
	if id == nil {
		return NullRequestID
	}

	return id.String()
}

type requestID struct {
	notNull    bool
	renderNull bool

	stringID *string
	intID    *int
	floatID  *float64
}

func (i *requestID) String() string {
	if !i.notNull {
		return NullRequestID
	}

	if i.stringID != nil {
		return fmt.Sprintf(`"%s"`, *i.stringID)
	}

	if i.floatID != nil {
		return fmt.Sprintf(`%v`, *i.floatID)
	}

	return fmt.Sprintf(`%v`, *i.intID)
}

func getRequestID(v *fastjson.Value) *requestID {
	if !v.Exists("id") {
		return &requestID{}
	}

	idValue := v.Get("id")

	intID, err := idValue.Int()
	if err != nil {
		floatID, err := idValue.Float64()
		if err != nil {
			bytesID, err := idValue.StringBytes()
			if err != nil {
				nullReqID := NullRequestID

				return &requestID{stringID: &nullReqID, notNull: true}
			}

			stringID := string(bytesID)

			return &requestID{stringID: &stringID, notNull: true}
		}

		return &requestID{floatID: &floatID, notNull: true}
	}

	return &requestID{intID: &intID, notNull: true}
}
