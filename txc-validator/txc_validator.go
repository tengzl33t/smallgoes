/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.

SPDX-License-Identifier: MPL-2.0

File: txc_validator.go
Description: TXCertbot 2.X compact Go config validator
Author: tengzl33t

Better to compile with tinygo:
GOTOOLCHAIN=go1.21.6 GOSUMDB='sum.golang.org' tinygo build -scheduler=none -panic=trap -gc=leaking ./txc_validator.go
*/

package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type StringSet map[string]struct{}

type siteGroupStruct struct {
	GroupName               string          `json:"group_name"`
	Sites                   []string        `json:"sites"`
	CertMode                string          `json:"cert_mode"`
	CertProvider            string          `json:"cert_provider"`
	CertType                string          `json:"cert_type"`
	CertProviderCredentials *EABCredentials `json:"cert_provider_creds"`
}

type tenantStruct struct {
	Tenant     string            `json:"tenant"`
	Env        string            `json:"env"`
	SiteGroups []siteGroupStruct `json:"site_groups"`
}

type EABCredentials struct {
	Email   string `json:"email"`
	KID     string `json:"kid"`
	HMACKey string `json:"hmac_key"`
}

func getAllowedProviders() []string {
	return []string{
		"letsencrypt",
		"buypass",
		"zerossl",
		"sslcom",
		"google",
		"google_test",
		"buypass_test",
		"letsencrypt_test",
	}
}

func getAllowedCertTypes() []string {
	return []string{
		"ec-256",
		"ec-384",
		"2048",
		"3072",
		"4096",
	}
}

func getAllowedCertModes() []string {
	return []string{
		"san",
		"classic",
	}
}

func getCertModeRegex() *regexp.Regexp {
	reg, _ := regexp.Compile("^\\*\\.\\S+\\.\\w+$")
	return reg
}

func getCertModeSiteRegex(certMode string) *regexp.Regexp {
	fixedMode := strings.ReplaceAll(certMode[2:], ".", "\\.")
	reg, _ := regexp.Compile("^([a-zA-Z0-9-]+\\.)?" + fixedMode + "$")
	return reg
}

func getSimpleSiteRegex() *regexp.Regexp {
	reg, _ := regexp.Compile("^\\S+\\.\\w+$")
	return reg
}

func validateCertMode(certMode string) bool {
	switch {
	case certMode == "":
		return true
	case slices.Contains(getAllowedCertModes(), certMode):
		return true
	case getCertModeRegex().MatchString(certMode):
		return true
	default:
		return false
	}
}

func validateSite(site string, certMode string) bool {
	switch {
	case getCertModeRegex().MatchString(certMode) && getCertModeSiteRegex(certMode).MatchString(site):
		return true
	case getSimpleSiteRegex().MatchString(site) && !getCertModeRegex().MatchString(certMode):
		return true
	default:
		return false
	}
}

func validateCertProvider(certProvider string) bool {
	switch {
	case certProvider == "":
		return true
	case slices.Contains(getAllowedProviders(), certProvider):
		return true
	default:
		return false
	}
}

func validateCertType(certType string) bool {
	switch {
	case certType == "":
		return true
	case slices.Contains(getAllowedCertTypes(), certType):
		return true
	default:
		return false
	}
}

func validateCertProviderCredentials(certProviderCredentials *EABCredentials) bool {
	switch {
	case certProviderCredentials == nil:
		return true
	case certProviderCredentials.Email != "" && certProviderCredentials.HMACKey != "" &&
		certProviderCredentials.KID != "":
		return true
	default:
		return false
	}
}

func prepareSGErrorMessage(checkType string, gotValue string, expectedValue string) string {
	return "Incorrect SG field '" + checkType + "' value: '" + gotValue + "'. Value must be one of: " + expectedValue
}

func validateSGs(sgs []siteGroupStruct, errors *[]string) {
	tenantSites := make(StringSet)

	for _, siteGroupObj := range sgs {
		if siteGroupObj.GroupName == "" {
			*errors = append(*errors, "SG field 'group_name' not found or empty")
		}

		if len(siteGroupObj.Sites) == 0 {
			*errors = append(*errors, "Field 'sites' not found or empty")
		} else {
			for _, site := range siteGroupObj.Sites {
				if _, ok := tenantSites[site]; !ok {
					tenantSites[site] = struct{}{}
				} else {
					*errors = append(*errors, "Duplicate found for site '"+site+"'")
				}
			}
		}

		if !validateCertMode(siteGroupObj.CertMode) {
			*errors = append(
				*errors,
				prepareSGErrorMessage(
					"cert_mode",
					siteGroupObj.CertMode,
					strings.Join(getAllowedCertModes(), ", ")+
						", or regex '"+getCertModeRegex().String()+"'",
				),
			)
		}
		if !validateCertProvider(siteGroupObj.CertProvider) {
			*errors = append(
				*errors,
				prepareSGErrorMessage(
					"cert_provider",
					siteGroupObj.CertProvider,
					strings.Join(getAllowedProviders(), ", "),
				),
			)
		}
		if !validateCertType(siteGroupObj.CertType) {
			*errors = append(
				*errors,
				prepareSGErrorMessage(
					"cert_type",
					siteGroupObj.CertType,
					strings.Join(getAllowedCertTypes(), ", "),
				),
			)
		}
		if !validateCertProviderCredentials(siteGroupObj.CertProviderCredentials) {
			*errors = append(
				*errors,
				"Field 'cert_provider_creds' has incorrect format",
			)
		}
		for _, site := range siteGroupObj.Sites {
			if !validateSite(site, siteGroupObj.CertMode) {
				*errors = append(
					*errors,
					"Incorrect site field value: '"+site+"'. Value must correspond to site regex: '"+
						getSimpleSiteRegex().String()+"' and cert_mode '"+siteGroupObj.CertMode+"'",
				)
			}
		}
	}
}

func validateTenants(tenants []tenantStruct, errors *[]string) {
	for _, tenantStructObj := range tenants {

		if tenantStructObj.Tenant == "" {
			*errors = append(*errors, "Field 'tenant' not found or empty")
		}
		if tenantStructObj.Env == "" {
			*errors = append(*errors, "Field 'env' not found or empty")
		}
		if len(tenantStructObj.SiteGroups) == 0 {
			*errors = append(*errors, "Field 'site_groups' not found or empty")
			continue
		}
		validateSGs(tenantStructObj.SiteGroups, errors)
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
		println("usage: validator <command> <args>")
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
