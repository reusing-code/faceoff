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
var posixCommand = sh.RunCmd("command", "-v")

func isToolInstalled(name string) bool {
	err := posixCommand(name)
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
	if !isToolInstalled("dep") {
		err := exec.Command("bash", "-c", "curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh").Run()
		if err != nil {
			log.Fatalf("Error installing dep: %s", err.Error())
			return err
		}
	}
	return exec.Command("dep", "ensure").Run()
}

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	mg.Deps(InstallDeps)
	fmt.Println("Building...")

	return cmd.Run()
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	mg.Deps(Build)
	fmt.Println("Installing...")
	return os.Rename("./MyApp", "/usr/bin/MyApp")
}

func InstallDeps() error {
	checkGopherJS()
	err := depEnsure()
	if err != nil {
		return err
	}
	return nil
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("MyApp")
}
