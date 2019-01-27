package transport

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var templateColored = "\x1b[%dm%s\x1b[0m"

func id() string {
	now := time.Now().UnixNano()
	hash := fmt.Sprintf("%08x", now&0xffffffff)
	return hash
	//color := (now&0xffff)%8 + 29
	//return fmt.Sprintf(templateColored, color, hash)
}

func scan(body io.ReadCloser) ([]byte, io.ReadCloser) {
	if body == nil {
		return nil, nil
	}

	var data, _ = ioutil.ReadAll(body)
	body.Close()
	return data, ioutil.NopCloser(bytes.NewReader(data))
}

func since(start time.Time) string {
	diff := time.Now().Sub(start)
	return strconv.FormatInt(int64(diff.Round(time.Millisecond)/time.Millisecond), 10)
}

func kvs2string(name string, kvs map[string][]string, ident string) string {
	if len(kvs) == 0 {
		return ""
	}

	var (
		r = make([]string, len(kvs))
		i = 0
	)

	for k, vs := range kvs {
		var vx = make([]string, len(vs))
		for i := range vs {
			vx[i] = strings.Replace(vs[i], "\n", "\n"+ident+ident, -1)
		}

		r[i] = ident + k + ": " + strings.Join(vx, ", ")
		i++
	}

	return "\n" + name + ":\n" + strings.Join(r, "\n")
}
