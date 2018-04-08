// +build mage

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/magefile/mage/sh"

	"github.com/magefile/mage/mg"
)

var Default = Build

var g0 = sh.RunCmd("go")

func isToolInstalled(name string) bool {
	_, err := exec.LookPath(name)
	if err != nil {
		return false
	} else {
		return true
	}
}

func checkGopherJS() {
	if !isToolInstalled("gopherjs") {
		g0("get", "-u", "github.com/gopherjs/gopherjs")
	}
}

func depEnsure() error {
	if isToolInstalled("dep") {
		exec.Command("dep", "ensure").Run()
	}
}

func Build() error {
	mg.Deps(InstallDeps)
	log.Println("Building...")

	// build server
	cmd := exec.Command("go", "build", "-v")
	cmd.Dir = "webserver"
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}

	// build client
	// @TODO non minified compilation for development
	cmd = exec.Command("gopherjs", "build", "-m")
	cmd.Dir = "client"
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func Install() error {
	mg.Deps(Build)
	fmt.Println("Installing...")
	os.MkdirAll("package", 0755)
	err := exec.Command("cp", "-ruf", "static", "templates", "package/").Run()
	if err != nil {
		return err
	}
	err = exec.Command("cp", "-ruf", "webserver/webserver", "package/").Run()
	if err != nil {
		return err
	}
	err = exec.Command("cp", "-ruf", "client/client.js", "client/client.js.map", "package/static/").Run()
	if err != nil {
		return err
	}

	//ignore error
	exec.Command("cp", "uf", "version.txt", "package/").Run()

	return nil
}

func InstallDeps() error {
	log.Println("Installing Deps...")
	checkGopherJS()
	err := depEnsure()
	if err != nil {
		return err
	}
	return nil
}

func InstallDepTool() error {
	if !isToolInstalled("dep") {
		err := exec.Command("bash", "-c", "curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh").Run()
		if err != nil {
			log.Fatalf("Error installing dep: %s", err.Error())
			return err
		}
	}
}

func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("package")
}
