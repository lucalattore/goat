package goat

import (
	"io/ioutil"
	"os"
	"strings"
)

// EvaluateEnv find and replace any environment variable in the provided string s.
// An env variable is specified as ${VARNAME}.
// A special case is ${file:/path/file} where the placeholder is replaced by the content
// of the specified file. This case is very useful to handle docker secrets the normaly
// are mounted into your container as file at the path like /run/secrets/my_secret.
func EvaluateEnv(s string) string {
	i := strings.Index(s, "${")
	if i >= 0 {
		j := strings.Index(s, "}")
		if j > i {
			x := s[i+2 : j]
			var y string
			if strings.HasPrefix(x, "file:") {
				x = x[5:]
				buf, err := ioutil.ReadFile(x)
				if err == nil {
					y = string(buf)
				}
			} else {
				y = os.Getenv(x)
			}
			s = s[:i] + y + EvaluateEnv(s[j+1:])
		}
	}

	return s
}
