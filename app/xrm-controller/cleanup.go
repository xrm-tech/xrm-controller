package xrmcontroller

import "bytes"

func bodyPasswordCleanup(body []byte) []byte {
	pos := bytes.Index(body, []byte("_password\""))
	if pos == -1 {
		return body
	}
	var buf bytes.Buffer
	buf.Grow(len(body))

	for pos != -1 {
		pos += 10
		buf.Write(body[:pos])

		body = body[pos:]
		pos = bytes.Index(body, []byte(":"))
		if pos == -1 {
			buf.Write(body)
		} else {
			buf.Write(body[:pos+1])
			body = body[pos+1:]
			buf.WriteString("\"<STRIPPED>")
			pos = bytes.Index(body, []byte("\""))
			if pos != -1 {
				body = body[pos+1:]
				pos = bytes.Index(body, []byte("\""))
			}
			if pos != -1 {
				buf.WriteByte('"')
				body = bytes.TrimLeft(body[pos+1:], " ")
				if len(body) > 0 && body[0] == ',' {
					buf.WriteByte(',')
					body = body[1:]
				}
				pos = bytes.Index(body, []byte("_password\""))
				if pos == -1 {
					buf.Write(body)
				}
			}
		}
	}

	return buf.Bytes()
}
