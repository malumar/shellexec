package shellexec

import (
	"fmt"
	"os/user"
)

type Environment = map[string]string

func NewEnvironmentFromProcess() (Environment, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	return NewEnvironment(currentUser.Name, currentUser.HomeDir), nil
}

func NewEnvironment(login, home string) Environment {
	return Environment{"LOGNAME": login, "HOME": home}
}

func EnvironmentToSliceOfStr(environment Environment) (ret []string) {
	for k, v := range environment {
		if len(k) == 0 {
			continue
		}
		ret = append(ret, fmt.Sprintf("%s=%s\n", k, v))
	}
	return
}
