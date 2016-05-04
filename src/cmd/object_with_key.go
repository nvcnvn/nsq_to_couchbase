package main

import (
	jlexer "github.com/mailru/easyjson/jlexer"
)

// use for parse the JSON string when default field name given
type parseObject struct {
	MessageID string `json:"messageId"`
}

func decode_parseObject(in *jlexer.Lexer, out *parseObject) {
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "messageId":
			out.MessageID = in.String()
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
}

func (p *parseObject) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	decode_parseObject(&r, p)
	return r.Error()
}
