package langs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ClojureLangHelper provides a set of helper methods for the lifecycle of Clojure Leinengen projects
type ClojureLangHelper struct {
	BaseHelper
}

// BuildFromImage returns the Docker image used to compile the Maven function project
func (lh *ClojureLangHelper) BuildFromImage() string { return "quay.io/markxnelson/fn-clojure-fdk-build:latest" }

// RunFromImage returns the Docker image used to run the Java function.
func (lh *ClojureLangHelper) RunFromImage() string {
	return "quay.io/markxnelson/fn-clojure-fdk-build:latest"
}

// HasBoilerplate returns whether the Clojure runtime has boilerplate that can be generated.
func (lh *ClojureLangHelper) HasBoilerplate() bool { return true }

// GenerateBoilerplate will generate function boilerplate for a Clojure runtime. The default boilerplate is for a Leinengen
// project.
func (lh *ClojureLangHelper) GenerateBoilerplate() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	pathToProjectFile := filepath.Join(wd, "project.clj")
	if exists(pathToProjectFile) {
		return ErrBoilerplateExists
	}

	apiVersion, err := getFDKAPIVersion()
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(pathToProjectFile, []byte(projectFileContent(apiVersion)), os.FileMode(0644)); err != nil {
		return err
	}

	mkDirAndWriteFile := func(dir, filename, content string) error {
		fullPath := filepath.Join(wd, dir)
		if err = os.MkdirAll(fullPath, os.FileMode(0755)); err != nil {
			return err
		}

		fullFilePath := filepath.Join(fullPath, filename)
		return ioutil.WriteFile(fullFilePath, []byte(content), os.FileMode(0644))
	}

	err = mkDirAndWriteFile("src", "hello.clj", helloClojureSrcBoilerplate)
	if err != nil {
		return err
	}

	return mkDirAndWriteFile("test", "hello_test.clj", helloClojureTestBoilerplate)
}

// Cmd returns the Clojure runtime Docker entrypoint that will be executed when the function is executed.
func (lh *ClojureLangHelper) Cmd() string {
	return "hello"
}

// DockerfileCopyCmds returns the Docker COPY command to copy the compiled Clojure function jar and dependencies.
func (lh *ClojureLangHelper) DockerfileCopyCmds() []string {
	return []string{
		"COPY --from=build-stage /function/target/*.jar /function/app/",
		"COPY --from=build-stage /function/src/* /function/src/",
		"COPY --from=build-stage /function/project.clj /function/",
	}
}

// DockerfileBuildCmds returns the build stage steps to compile the Maven function project.
func (lh *ClojureLangHelper) DockerfileBuildCmds() []string {
	return []string{
		fmt.Sprintf("ENV LEIN_OPTS %s", leinOpts()),
		"ADD project.clj /function/project.clj",
		"RUN [\"lein\", \"deps\"]",
		"ADD src /function/src",
		"RUN [\"lein\", \"uberjar\"]",
	}
}

// HasPreBuild returns whether the Java Maven runtime has a pre-build step.
func (lh *ClojureLangHelper) HasPreBuild() bool { return true }

// PreBuild ensures that the expected the function is based is a maven project.
func (lh *ClojureLangHelper) PreBuild() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if !exists(filepath.Join(wd, "project.clj")) {
		return errors.New("Could not find project.clj - are you sure this is a Leinengen project?")
	}

	return nil
}

func leinOpts() string {
	var opts bytes.Buffer

	if parsedURL, err := url.Parse(os.Getenv("http_proxy")); err == nil {
		opts.WriteString(fmt.Sprintf("-Dhttp.proxyHost=%s ", parsedURL.Hostname()))
		opts.WriteString(fmt.Sprintf("-Dhttp.proxyPort=%s ", parsedURL.Port()))
	}

	if parsedURL, err := url.Parse(os.Getenv("https_proxy")); err == nil {
		opts.WriteString(fmt.Sprintf("-Dhttps.proxyHost=%s ", parsedURL.Hostname()))
		opts.WriteString(fmt.Sprintf("-Dhttps.proxyPort=%s ", parsedURL.Port()))
	}

	nonProxyHost := os.Getenv("no_proxy")
	opts.WriteString(fmt.Sprintf("-Dhttp.nonProxyHosts=%s ", strings.Replace(nonProxyHost, ",", "|", -1)))

	//opts.WriteString("-Dmaven.repo.local=/usr/share/maven/ref/repository")

	return opts.String()
}

/*    TODO temporarily generate lein project boilerplate from hardcoded values.
Will eventually move to using a maven archetype.
*/
func projectFileContent(version string) string {
	return fmt.Sprintf(projectFile) //, version, version)
}

//func getFDKAPIVersion() (string, error) {`
//	const versionURL = "https://api.bintray.com/search/packages/maven?repo=fnproject&g=com.fnproject.fn&a=fdk"
//	const versionEnv = "FN_CLOJURE_FDK_VERSION"
//	fetchError := fmt.Errorf("Failed to fetch latest Clojure FDK version from %v. Check your network settings or manually override the version by setting %s", versionURL, versionEnv)
//
//	type parsedResponse struct {
//		Version string `json:"latest_version"`
//	}
//	version := os.Getenv(versionEnv)
//	if version != "" {
//		return version, nil
//	}
//	resp, err := http.Get(versionURL)
//	if err != nil || resp.StatusCode != 200 {
//		return "", fetchError
//	}
//
//	buf := bytes.Buffer{}
//	_, err = buf.ReadFrom(resp.Body)
//	if err != nil {
//		return "", fetchError
//	}
//
//	parsedResp := make([]parsedResponse, 1)
//	err = json.Unmarshal(buf.Bytes(), &parsedResp)
//	if err != nil {
//		return "", fetchError
//	}
//	return parsedResp[0].Version, nil
//}

const (
	projectFile = `(defproject hello "0.1.0-SNAPSHOT"
  :description "FIXME: write description"
  :url "http://example.com/FIXME"
  :main hello
  :dependencies [[org.clojure/clojure "1.8.0"]])
`

	helloClojureSrcBoilerplate = `(ns hello
	(:gen-class))

(defn hello
  "I don't do a whole lot."
  [x]
  (println "Hello, " x "!"))

(defn -main [& args]
  (hello (first args)))
`

	helloClojureTestBoilerplate = `(ns hello
  (:require [clojure.test :refer :all]
            [test.core :refer :all]))

(deftest hello-test
  (testing "FIXME, I do nothing."
    (is (= 0 0))))
`
)
