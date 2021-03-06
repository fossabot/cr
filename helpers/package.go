package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"

	"github.com/sunshinekitty/cr/models"
)

var (
	match = regexp.MustCompile

	packageName = match(`([a-z\d]){1}([a-z0-9-*_*]){0,48}([a-z\d]){1}`)
	repoName    = match(`([A-Za-z\d\./:-]*){3,141}`)

	// ErrInvalidPackageName is thrown when an invalid package name is given
	ErrInvalidPackageName = errors.New("package name is invalid")
	// ErrInvalidRepositoryName is thrown when an invalid repository name is given
	ErrInvalidRepositoryName = errors.New("repository name is invalid")
	// ErrInvalidPort is thrown when an invalid port is given
	ErrInvalidPort = errors.New("port number is invalid")
	// ErrInvalidVolume is thrown when an invalid volume mount is given
	ErrInvalidVolume = errors.New("volume is invalid")
	// ErrLongShortDescription is thrown when short description is too long (>200)
	ErrLongShortDescription = errors.New("short description is too long (>200 chars)")
	// ErrLongLongDescription is thrown when long description is too long (>25000)
	ErrLongLongDescription = errors.New("long description is too long (>25000 chars)")
	// ErrLongHomepage is thrown when home page is too long (>100)
	ErrLongHomepage = errors.New("homepage is too long (>100 chars)")
	// ErrLongCommandStart is thrown when command start is too long (>100)
	ErrLongCommandStart = errors.New("command start is too long (>100 chars)")
	// ErrMissingUsername is thrown when a username isn't set in client config
	ErrMissingUsername = errors.New("username is not set in client config")
)

// ConfigFileToCmd takes a path to a crackle package config and outputs a
// docker command and args to run said package.
func ConfigFileToCmd(path string) (string, string, error) {
	var cmdBuff bytes.Buffer

	pt, err := ConfigFileToPackageToml(path)
	if err != nil {
		return "", "", err
	}

	cmdStart := ""
	if pt.CommandStart != nil {
		cmdStart = " " + *pt.CommandStart
	}

	cmdBuff.WriteString("docker run -t --rm ")

	for _, p := range pt.Ports {
		cmdBuff.WriteString(fmt.Sprintf("-p %s:%s ", p.Local, p.Container))
	}

	for _, v := range pt.Volumes {
		cmdBuff.WriteString(fmt.Sprintf("-v %s:%s ", v.Local, v.Container))
	}

	cmdBuff.WriteString(fmt.Sprintf("%s%s", pt.Repository, cmdStart))

	return "/usr/bin/env", cmdBuff.String(), nil
}

// ConfigFileToPackageToml takes a path to toml config and translates to PackageToml struct
func ConfigFileToPackageToml(path string) (*models.PackageToml, error) {
	var returnPackageToml models.PackageToml
	_, err := toml.DecodeFile(path, &returnPackageToml)
	return &returnPackageToml, err
}

// PackageTomlToPackage takes a PackageToml struct and converts it to a Package struct
func PackageTomlToPackage(pt *models.PackageToml) (*models.Package, error) {
	splitRepository := strings.Split(pt.Repository, ":")
	username := viper.GetString("crackle.auth.username")
	if len(username) == 0 {
		return nil, ErrMissingUsername
	}
	p := &models.Package{
		CommandStart:     pt.CommandStart,
		Homepage:         pt.Homepage,
		LongDescription:  pt.LongDescription,
		Name:             pt.Package,
		Pulls:            0,
		ShortDescription: pt.ShortDescription,
		Version:          splitRepository[1],
		Repository:       splitRepository[0],
		Owner:            username,
	}

	ptPorts, err := json.Marshal(pt.Ports)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(ptPorts, &p.Ports)
	if err != nil {
		return nil, err
	}

	ptVolumes, err := json.Marshal(pt.Volumes)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(ptVolumes, &p.Volumes)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// PackageToPackageToml converts a Package object to PackageToml object
func PackageToPackageToml(p *models.Package) (*models.PackageToml, error) {
	pt := &models.PackageToml{
		CommandStart:     p.CommandStart,
		Homepage:         p.Homepage,
		LongDescription:  p.LongDescription,
		Package:          p.Name,
		ShortDescription: p.ShortDescription,
		Repository:       fmt.Sprintf("%s:%s", p.Repository, p.Version),
	}

	pPorts, err := json.Marshal(p.Ports)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(pPorts, &pt.Ports)
	if err != nil {
		return nil, err
	}

	pVolumes, err := json.Marshal(p.Volumes)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(pVolumes, &pt.Volumes)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

// ValidPackageToml validates a PackageToml object
func ValidPackageToml(pt *models.PackageToml) error {
	if !ValidPackageName(pt.Package) {
		return ErrInvalidPackageName
	}
	if !ValidRepositoryName(pt.Repository) {
		return ErrInvalidRepositoryName
	}
	for _, port := range pt.Ports {
		if !ValidPort(port.Container) {
			ErrInvalidPort = fmt.Errorf("Container port \"%v\" is invalid", port.Container)
			return ErrInvalidPort
		}
		if !ValidPort(port.Local) {
			ErrInvalidPort = fmt.Errorf("Local port \"%v\" is invalid", port.Local)
			return ErrInvalidPort
		}
	}
	for _, volume := range pt.Volumes {
		if len(volume.Container) > 4351 {
			ErrInvalidVolume = fmt.Errorf("Container volume \"%v\" is too long", volume.Container)
			return ErrInvalidVolume
		}
		if len(volume.Local) > 4351 {
			ErrInvalidVolume = fmt.Errorf("Local volume \"%v\" is too long", volume.Local)
			return ErrInvalidVolume
		}
	}
	if pt.ShortDescription != nil {
		if len(fmt.Sprintf("%s", *pt.ShortDescription)) > 200 {
			return ErrLongShortDescription
		}
	}
	if pt.LongDescription != nil {
		if len(fmt.Sprintf("%s", *pt.LongDescription)) > 25000 {
			return ErrLongLongDescription
		}
	}
	if pt.Homepage != nil {
		if len(fmt.Sprintf("%s", *pt.Homepage)) > 100 {
			return ErrLongHomepage
		}
	}
	if pt.CommandStart != nil {
		if len(fmt.Sprintf("%s", *pt.CommandStart)) > 100 {
			return ErrLongCommandStart
		}
	}
	return nil
}

// ValidPackage validates a Package object
func ValidPackage(p *models.Package) error {
	if !ValidPackageName(p.Name) {
		return ErrInvalidPackageName
	}
	if !ValidRepositoryName(fmt.Sprintf("%s:%s", p.Repository, p.Version)) {
		return ErrInvalidRepositoryName
	}

	portsBytes, err := json.Marshal(p.Ports)
	ports := new(models.Ports)
	if err != nil {
		return err
	}
	err = json.Unmarshal(portsBytes, &ports)
	for _, port := range *ports {
		if !ValidPort(port.Container) {
			ErrInvalidPort = fmt.Errorf("Container port \"%v\" is invalid", port.Container)
			return ErrInvalidPort
		}
		if !ValidPort(port.Local) {
			ErrInvalidPort = fmt.Errorf("Local port \"%v\" is invalid", port.Local)
			return ErrInvalidPort
		}
	}

	volumesBytes, err := json.Marshal(p.Volumes)
	volumes := new(models.Volumes)
	if err != nil {
		return err
	}
	err = json.Unmarshal(volumesBytes, &volumes)
	for _, volume := range *ports {
		if len(volume.Container) > 4351 {
			ErrInvalidVolume = fmt.Errorf("Container volume \"%v\" is too long", volume.Container)
			return ErrInvalidVolume
		}
		if len(volume.Local) > 4351 {
			ErrInvalidVolume = fmt.Errorf("Local volume \"%v\" is too long", volume.Local)
			return ErrInvalidVolume
		}
	}

	if p.ShortDescription != nil {
		if len(fmt.Sprintf("%s", *p.ShortDescription)) > 200 {
			return ErrLongShortDescription
		}
	}
	if p.LongDescription != nil {
		if len(fmt.Sprintf("%s", *p.LongDescription)) > 25000 {
			return ErrLongLongDescription
		}
	}
	if p.Homepage != nil {
		if len(fmt.Sprintf("%s", *p.Homepage)) > 100 {
			return ErrLongHomepage
		}
	}
	if p.CommandStart != nil {
		if len(fmt.Sprintf("%s", *p.CommandStart)) > 100 {
			return ErrLongCommandStart
		}
	}

	return nil
}

// ValidPackageName validates a package's name
func ValidPackageName(n string) bool {
	if len(n) == 0 {
		return false
	}
	return len(packageName.FindString(n)) == len(n)
}

// ValidRepositoryName validates a repository name
func ValidRepositoryName(n string) bool {
	// We could pull in Docker and use their regexp matching, but I don't think it really matters
	// We should just verify it meets database constraints and is alphanumeric and/or ":" and/or "/"'s
	if len(n) > 141 || len(n) < 3 {
		return false
	}
	return len(repoName.FindString(n)) == len(n)
}

// ValidPort validate's a port number
func ValidPort(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return i >= 1 && i <= 65535
}
