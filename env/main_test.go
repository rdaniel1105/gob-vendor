package env

import (
	"fmt"
	"math"
	"math/rand"
	"os"
)

func withTestEnv(val interface{}, cb func(varName string)) {
	varName := fmt.Sprintf("TEST_%d_%d", rand.Intn(math.MaxInt32), rand.Intn(math.MaxInt32))
	_ = os.Setenv(varName, fmt.Sprintf("%v", val))

	defer func() {
		_ = os.Unsetenv(varName)
	}()

	cb(varName)
}
