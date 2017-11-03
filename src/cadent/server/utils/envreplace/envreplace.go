/**
 given a string replace any entities that match
	$ENV{MY_ENV_VAR:defaultvalue}
  from the shell environment
*/

package envreplace

import (
	"bytes"
	"os"
	"regexp"
)

var envReg *regexp.Regexp

func init() {
	envReg = regexp.MustCompile(`\$ENV\{(.*?)\}`)
}

func ReplaceEnv(inbys []byte) []byte {

	// ye-old-replace
	for _, mtch := range envReg.FindAllSubmatch(inbys, -1) {
		if len(mtch) != 2 {
			continue
		}
		if len(mtch[0]) == 0 {
			continue
		}

		inbtween := bytes.Split(mtch[1], []byte(":"))
		envvar := string(inbtween[0])
		def := []byte("")
		if len(inbtween) >= 2 {
			def = inbtween[1]
		}
		env := os.Getenv(envvar)
		if len(env) > 0 {
			inbys = bytes.Replace(inbys, mtch[0], []byte(env), -1)
		} else {
			inbys = bytes.Replace(inbys, mtch[0], def, -1)
		}
	}
	return inbys
}
