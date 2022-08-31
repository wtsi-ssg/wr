/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

// Package user is used to get the user specific details like uid, username, homedir etc.
package user

import (
	"os/user"
	"strconv"
)

// userName returns the Username of the current user.
func userName() (string, error) {
	username, err := getUserDetails("username")
	if err != nil {
		return "", err
	}

	return username, nil
}

// userID returns the ID of the current user.
func userID() (int, error) {
	uidStr, err := getUserDetails("id")
	if err != nil {
		return 0, err
	}

	userid, err := strconv.Atoi(uidStr)
	if err != nil {
		return 0, err
	}

	return userid, nil
}

// getUserDetails defines a switch case to include all the possible user specific details from os/user package.
func getUserDetails(idopt string) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	switch idopt {
	case "id":
		return user.Uid, err
	case "username":
		return user.Username, err
	default:
		return "", err
	}
}
