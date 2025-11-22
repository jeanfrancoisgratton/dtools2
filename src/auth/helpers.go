// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 21:13
// Original filename: src/auth/helpers.go

package auth

import (
	"os/user"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func GetUser() (string, *ce.CustomError) {
	cUsr, err := user.Current()
	if err != nil {
		return "", &ce.CustomError{Title: "Unable to find user", Message: err.Error()}
	}

	return cUsr.Username, nil
}
