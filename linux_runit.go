// Copyright 2014 Igor Dolzhikov. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

package daemon

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"text/template"
  "path"
)

// systemDRecord - standard record (struct) for linux systemD version of daemon package
type runitRecord struct {
	name        string
	description string
}

// Standard service path for systemD daemons
func (linux *runitRecord) servicePath() string {
	return "/etc/service/" + linux.name
}

// Check service is installed
func (linux *runitRecord) checkInstalled() bool {

	if _, err := os.Stat(path.Join(linux.servicePath(),"run")); err == nil {
		return true
	}

	return false
}

// Check service is running
func (linux *runitRecord) checkRunning() (string, bool) {
	output, err := exec.Command("sv", "status", linux.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString("run: ", string(output)); err == nil && matched {
			reg := regexp.MustCompile("pid ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return "Service (pid  " + data[1] + ") is running...", true
			}
			return "Service is running...", true
		}
	}

	return "Service is stoped", false
}

// Install the service
func (linux *runitRecord) Install() (string, error) {
	installAction := "Install " + linux.description + ":"

	if checkPrivileges() == false {
		return installAction + failed, errors.New(rootPrivileges)
	}

	srvPath := linux.servicePath()

	if linux.checkInstalled() == true {
		return installAction + failed, errors.New(linux.description + " already installed")
	}

  if err:=os.MkdirAll(srvPath, 0755); err!= nil {
		return installAction + failed, err
  }
	file, err := os.Create(path.Join(srvPath,"run"))
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(linux.name)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("runitConfig").Parse(runitConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Path string
		}{linux.name, linux.description, execPatch},
	); err != nil {
		return installAction + failed, err
	}

  file.Chmod(0755)
  /*
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return installAction + failed, err
	}

	if err := exec.Command("systemctl", "enable", linux.name+".service").Run(); err != nil {
		return installAction + failed, err
	}

  */

	return installAction + success, nil
}

// Remove the service
func (linux *runitRecord) Remove() (string, error) {
	removeAction := "Removing " + linux.description + ":"

	if checkPrivileges() == false {
		return removeAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return removeAction + failed, errors.New(linux.description + " is not installed")
	}

	if err := exec.Command("sv", "stop", linux.name).Run(); err != nil {
		return removeAction + failed, err
	}

	if err := os.RemoveAll(path.Join(linux.servicePath())); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (linux *runitRecord) Start() (string, error) {
	startAction := "Starting " + linux.description + ":"

	if checkPrivileges() == false {
		return startAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return startAction + failed, errors.New(linux.description + " is not installed")
	}

	if _, status := linux.checkRunning(); status == true {
		return startAction + failed, errors.New("service already running")
	}

	if err := exec.Command("sv", "start", linux.name).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (linux *runitRecord) Stop() (string, error) {
	stopAction := "Stopping " + linux.description + ":"

	if checkPrivileges() == false {
		return stopAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return stopAction + failed, errors.New(linux.description + " is not installed")
	}

	if _, status := linux.checkRunning(); status == false {
		return stopAction + failed, errors.New("service already stopped")
	}

	if err := exec.Command("sv", "stop", linux.name).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (linux *runitRecord) Status() (string, error) {

	if checkPrivileges() == false {
		return "", errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return "Status could not defined", errors.New(linux.description + " is not installed")
	}

	statusAction, _ := linux.checkRunning()

	return statusAction, nil
}

var runitConfig = `#!/bin/bash
#{{.Description}}

exec {{.Path}} >>/var/log/{{.Name}}.log 2>&1
`
