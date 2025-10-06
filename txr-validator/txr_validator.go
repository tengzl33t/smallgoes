/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.

SPDX-License-Identifier: MPL-2.0

File: txr_validator.go
Description: TXReports 2.X compact Go config validator
Author: tengzl33t

Better to compile with tinygo:
GOTOOLCHAIN=go1.21.6 GOSUMDB='sum.golang.org' tinygo build -scheduler=none -panic=trap -gc=leaking ./txr_validator.go
*/

package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type scheduleStruct struct {
	DaysToCollect int    `json:"days_to_collect"`
	CronToSend    string `json:"cron_to_send"`
}

type featureStruct struct {
	Rules    *bool `json:"rules"`
	Entities *bool `json:"entities"`
	Stats    *bool `json:"stats"`
	Matches  *bool `json:"matches"`
	Blocks   *bool `json:"blocks"`
}

type tenantStruct struct {
	Tenant    string           `json:"tenant"`
	LogoPath  *string          `json:"logo_path"`
	Emails    []string         `json:"emails"`
	Schedules []scheduleStruct `json:"schedules"`
	Features  *featureStruct   `json:"features"`
}

func getLogoRegex() *regexp.Regexp {
	return regexp.MustCompile(`.+\.(?:jpg|jpeg|png|svg|tiff|tif|webp|gif|bmp|avif|jfif)$`)
}

func getCronRegex() *regexp.Regexp {
	return regexp.MustCompile(`^((((\d+,)+\d+|(\d+([/\-])\d+)|\d+|\*(/\d+)?|[A-Z]{3}(-[A-Z]{3})?) ?){5,7})$`)
}

func checkEmails(emails []string) bool {
	for _, email := range emails {
		if !strings.Contains(email, "@") {
			return false
		}
	}
	return true
}

func validateSchedules(schedules []scheduleStruct, errors *[]string) {
	for _, schedule := range schedules {
		if schedule.DaysToCollect > 366 || schedule.DaysToCollect < 1 {
			*errors = append(*errors, "Field 'days_to_collect' can't be greater than 366 or less than 1")
		}
		if !getCronRegex().MatchString(schedule.CronToSend) {
			*errors = append(*errors, "Field 'cron_to_send' has incorrect format")
		}
	}
}

func validateTenants(tenants []tenantStruct, errors *[]string) {
	for _, tenantStructObj := range tenants {

		if tenantStructObj.Tenant == "" {
			*errors = append(*errors, "Field 'tenant' not found or empty")
		}
		if tenantStructObj.LogoPath != nil && !getLogoRegex().MatchString(*tenantStructObj.LogoPath) {
			*errors = append(*errors, "Field 'logo_path' does not match the expected format")
		}
		if len(tenantStructObj.Emails) == 0 {
			*errors = append(*errors, "Field 'emails' not found or empty")
		} else {
			if !checkEmails(tenantStructObj.Emails) {
				*errors = append(*errors, "Field 'emails' has incorrect entries")
			}
		}
		if len(tenantStructObj.Schedules) == 0 {
			*errors = append(*errors, "Field 'schedules' not found or empty")
			continue
		}

		validateSchedules(tenantStructObj.Schedules, errors)

	}
}

func getFileOrDir(path string) []string {
	file, err := os.Stat(path)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	switch {
	case file.Mode().IsDir():
		if path[len(path)-1] != '/' {
			path = path + "/"
		}
		jsonFiles, _ := filepath.Glob(path + "*.json")
		return jsonFiles
	case file.Mode().IsRegular():
		fileExt := path[strings.LastIndex(path, ".")+1:]

		if fileExt != "json" {
			println(err)
			os.Exit(1)
		}
		return []string{path}
	default:
		return []string{}
	}
}

func main() {
	cmdArgs := os.Args[1:]
	if len(cmdArgs) < 1 {
		println("usage: validator <file or dir path>")
		os.Exit(1)
	}

	var jsonFilePaths []string

	for _, cmdArg := range cmdArgs {
		jsonFilePaths = append(jsonFilePaths, getFileOrDir(cmdArg)...)
	}

	if len(jsonFilePaths) == 0 {
		println("No JSON files found.")
		os.Exit(1)
	}

	var errors []string

	for _, jsonFilePath := range jsonFilePaths {
		var runErrors []string
		jsonFile, err := os.Open(jsonFilePath)
		if err != nil {
			println("No such file or directory: " + jsonFilePath)
			os.Exit(1)
		}

		var tenantStructs []tenantStruct

		byteValue, _ := io.ReadAll(jsonFile)

		err = json.Unmarshal(byteValue, &tenantStructs)
		if err != nil {
			runErrors = append(
				runErrors,
				"Incorrect JSON format: "+err.Error(),
			)
		}

		validateTenants(tenantStructs, &runErrors)

		if len(runErrors) > 0 {
			errorMsgPart := "Config file '" + jsonFilePath + "' validation failed.\nIssues:\n" +
				"- " + strings.Join(runErrors, "\n- ")
			errors = append(errors, errorMsgPart)
		} else {
			println("Config file " + jsonFilePath + " validation succeeded.")
		}

		_ = jsonFile.Close()
	}

	if len(errors) > 0 {
		for _, err := range errors {
			println(err)
		}
		os.Exit(1)
	}

	os.Exit(0)

}
