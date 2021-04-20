/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>, Ashwini Chhipa <ac55@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 ******************************************************************************/

package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/creasty/defaults"
	"github.com/jinzhu/configor"
	"github.com/olekukonko/tablewriter"
	"github.com/wtsi-ssg/wr/clog"
)

const (
	// configCommonBasename is the basename of a wr config file.
	configCommonBasename = ".wr_config.yml"

	// S3Prefix is the prefix used by S3 paths.
	S3Prefix = "s3://"

	// Production is the name of the main deployment.
	Production = "production"

	// Development is the name of the development deployment, used during
	// testing.
	Development = "development"

	// ConfigSourceEnvVar is a config value source.
	ConfigSourceEnvVar = "env var"

	// ConfigSourceDefault is a config value source.
	ConfigSourceDefault = "default"

	// sourcesProperty is a source property.
	sourcesProperty = "sources"

	// maxPort is the maximum port available.
	maxPort = 65535

	// minport is the minimum port used.
	minPort = 1021

	// portsPerUser is the number of ports used for each user.
	portsPerUser = 4
)

// Config holds the configuration options for jobqueue server and client.
type Config struct {
	ManagerPort          string `default:""`
	ManagerWeb           string `default:""`
	ManagerHost          string `default:"localhost"`
	ManagerDir           string `default:"~/.wr"`
	ManagerPidFile       string `default:"pid"`
	ManagerLogFile       string `default:"log"`
	ManagerDBFile        string `default:"db"`
	ManagerDBBkFile      string `default:"db_bk"`
	ManagerTokenFile     string `default:"client.token"`
	ManagerUploadDir     string `default:"uploads"`
	ManagerUmask         int    `default:"007"`
	ManagerScheduler     string `default:"local"`
	ManagerCAFile        string `default:"ca.pem"`
	ManagerCertFile      string `default:"cert.pem"`
	ManagerKeyFile       string `default:"key.pem"`
	ManagerCertDomain    string `default:"localhost"`
	ManagerSetDomainIP   bool   `default:"false"`
	RunnerExecShell      string `default:"bash"`
	Deployment           string `default:"production"`
	CloudFlavor          string `default:""`
	CloudFlavorManager   string `default:""`
	CloudFlavorSets      string `default:""`
	CloudKeepAlive       int    `default:"120"`
	CloudServers         int    `default:"-1"`
	CloudCIDR            string `default:"192.168.0.0/18"`
	CloudGateway         string `default:"192.168.0.1"`
	CloudDNS             string `default:"8.8.4.4,8.8.8.8"`
	CloudOS              string `default:"bionic-server"`
	ContainerImage       string `default:"ubuntu:latest"`
	CloudUser            string `default:"ubuntu"`
	CloudRAM             int    `default:"2048"`
	CloudDisk            int    `default:"1"`
	CloudScript          string `default:""`
	CloudConfigFiles     string `default:"~/.s3cfg,~/.aws/credentials,~/.aws/config"`
	CloudSpawns          int    `default:"10"`
	CloudAutoConfirmDead int    `default:"30"`
	DeploySuccessScript  string `default:""`
	sources              map[string]string
}

// FileExistsError records a file exist error.
type FileExistsError struct {
	Path string
	Err  error
}

// Error returns error when a file already exists.
func (f *FileExistsError) Error() string {
	return fmt.Sprintf("file [%s] already exists: %s", f.Path, f.Err)
}

// Source returns where the value of a Config field was defined.
func (c Config) Source(field string) string {
	if c.sources == nil {
		return ConfigSourceDefault
	}

	source, set := c.sources[field]
	if !set {
		return ConfigSourceDefault
	}

	return source
}

// IsProduction tells you if we're in the production deployment.
func (c Config) IsProduction() bool {
	return c.Deployment == Production
}

// String retruns the string value of property.
func (c Config) String() string {
	vals := reflect.ValueOf(c)
	typeOfC := vals.Type()

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Config", "Value", "Source"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for i := 0; i < vals.NumField(); i++ {
		property := typeOfC.Field(i).Name
		if property == sourcesProperty {
			continue
		}

		source := c.sources[property]
		if source == "" {
			source = ConfigSourceDefault
		}

		table.Append([]string{property, fmt.Sprintf("%v", vals.Field(i).Interface()), source})
	}

	table.Render()

	return tableString.String()
}

// ConfigLoadFromParentDir loads and returns the config from a parent directory.
func ConfigLoadFromParentDir(ctx context.Context, deployment string) *Config {
	pwd := GetPWD(ctx)
	pwd = filepath.Dir(pwd)

	uid := os.Getuid()

	return mergeAllConfigs(ctx, uid, deployment, pwd, true)
}

// ConfigLoadFromParentDir loads and returns the config from a non-parent directory.
func ConfigLoadFromNonParentDir(ctx context.Context, deployment string) *Config {
	pwd := GetPWD(ctx)

	uid := os.Getuid()

	return mergeAllConfigs(ctx, uid, deployment, pwd, false)
}

// mergeAllConfigs function loads and merges all the configs and returns a final config.
func mergeAllConfigs(ctx context.Context, uid int, deployment string, pwd string, useparentdir bool) *Config {
	// if deployment not set on the command line
	if deployment != Development && deployment != Production {
		deployment = DefaultDeployment(ctx)
	}

	// load and merge default and env vars configs
	configDef := mergeDefaultAndEnvVarsConfigs(ctx)

	// read all config files and merge them
	configDef.mergeAllConfigFiles(ctx, uid, deployment, pwd, useparentdir)

	return configDef
}

// DefaultDeployment works out the default deployment.
func DefaultDeployment(ctx context.Context) string {
	pwd := GetPWD(ctx)

	// if we're in the git repository
	var deployment string

	_, err := os.Stat(filepath.Join(pwd, "jobqueue", "server.go"))
	if err == nil {
		// force development
		deployment = Development
	} else {
		// default to production
		deployment = Production
	}

	// and allow env var to override with development
	deploymentEnv := os.Getenv("WR_DEPLOYMENT")
	if deploymentEnv != "" {
		if deploymentEnv == Development {
			deployment = Development
		}
	}

	return deployment
}

// mergeDefaultAndEnvVarsConfigs returns a merged Default and EnvVar config.
func mergeDefaultAndEnvVarsConfigs(ctx context.Context) *Config {
	// load default config before setting env vars using configor
	configDef := loadDefaultConfig(ctx)

	// set ManagerUmask
	setenvManagerUmask()

	// load env var using configor
	configEnv := getEnvVarsConfig(ctx)

	// merge default and env var config
	configDef.merge(configEnv, ConfigSourceEnvVar)

	return configDef
}

// getDefaultConfig loads and return the default configs.
func loadDefaultConfig(ctx context.Context) *Config {
	// because we want to know the source of every value, we can't take
	// advantage of configor.Load() being able to take all env vars and config
	// files at once. We do it repeatedly and merge results instead
	config := &Config{}
	if cerr := defaults.Set(config); cerr != nil {
		clog.Fatal(ctx, cerr.Error())
	}

	return config
}

// setenvManagerUmask sets the WR_MANAGERUMASK env variable.
func setenvManagerUmask() {
	// load env vars. ManagerUmask is likely to be zero prefixed by user, but
	// that is not converted to int correctly, so fix first
	umask := os.Getenv("WR_MANAGERUMASK")
	if umask != "" && strings.HasPrefix(umask, "0") {
		umask = strings.TrimLeft(umask, "0")
		os.Setenv("WR_MANAGERUMASK", umask)
	}
}

// getEnvVarsConfig loads the env variables and returns the config.
func getEnvVarsConfig(ctx context.Context) *Config {
	// we don't os.Setenv("CONFIGOR_ENV", deployment) to stop configor loading
	// files before we want it to
	err := os.Setenv("CONFIGOR_ENV_PREFIX", "WR")
	if err != nil {
		clog.Fatal(ctx, err.Error())
	}

	configEnv := &Config{}

	err = configor.Load(configEnv)
	if err != nil {
		clog.Fatal(ctx, err.Error())
	}

	return configEnv
}

// merge compares existing to new Config values, and for each one that has
// changed, sets the given source on the changed property in our sources,
// and sets the new value on ourselves.
func (c *Config) merge(new *Config, source string) {
	v := reflect.ValueOf(*c)
	typeOfC := v.Type()
	vNew := reflect.ValueOf(*new)

	if c.sources == nil {
		c.sources = make(map[string]string)
	}

	for i := 0; i < v.NumField(); i++ {
		property := typeOfC.Field(i).Name
		if property == sourcesProperty {
			continue
		}

		if vNew.Field(i).Interface() != v.Field(i).Interface() {
			c.sources[property] = source

			adrField := reflect.ValueOf(c).Elem().Field(i)
			setSourceOnChangeProp(typeOfC, adrField, vNew, i)
		}
	}
}

// mergeAllConfigFiles merges all the config files and adjusts config properties.
func (c *Config) mergeAllConfigFiles(ctx context.Context, uid int, deployment string, pwd string, useparentdir bool) {
	configDeploymentBasename := ".wr_config." + deployment + ".yml"

	if configDir := os.Getenv("WR_CONFIG_DIR"); configDir != "" {
		c.configLoadFromFile(ctx, filepath.Join(configDir, configCommonBasename))
		c.configLoadFromFile(ctx, filepath.Join(configDir, configDeploymentBasename))
	}

	if useparentdir {
		home, herr := os.UserHomeDir()
		if herr != nil || home == "" {
			errStr := "could not find home dir"
			clog.Fatal(ctx, errStr)
		}

		c.configLoadFromFile(ctx, filepath.Join(home, configCommonBasename))
		c.configLoadFromFile(ctx, filepath.Join(home, configDeploymentBasename))
	}

	c.configLoadFromFile(ctx, filepath.Join(pwd, configCommonBasename))
	c.configLoadFromFile(ctx, filepath.Join(pwd, configDeploymentBasename))

	// adjust config properties and return
	c.adjustConfigProperties(ctx, uid, deployment)
}

// configLoadFromFile loads a config from a file and merges into the current config.
func (c *Config) configLoadFromFile(ctx context.Context, path string) {
	_, err := os.Stat(path)
	if err != nil {
		return
	}

	configFile := c.clone()

	err = configor.Load(configFile, path)
	if err != nil {
		clog.Fatal(ctx, err.Error())
	}

	c.merge(configFile, path)
}

// clone makes a new Config with our values.
func (c *Config) clone() *Config {
	new := &Config{}

	v := reflect.ValueOf(*c)
	typeOfC := v.Type()

	for i := 0; i < v.NumField(); i++ {
		property := typeOfC.Field(i).Name
		if property == sourcesProperty {
			continue
		}

		adrField := reflect.ValueOf(new).Elem().Field(i)
		setSourceOnChangeProp(typeOfC, adrField, v, i)
	}

	new.sources = make(map[string]string)
	for key, val := range c.sources {
		new.sources[key] = val
	}

	return new
}

// setSourceOnChangeProp sets the source of a property, when its value is changed.
func setSourceOnChangeProp(typeOfC reflect.Type, adrField reflect.Value, newVal reflect.Value, idx int) {
	switch typeOfC.Field(idx).Type.Kind() {
	case reflect.String:
		adrField.SetString(newVal.Field(idx).String())
	case reflect.Int:
		adrField.SetInt(newVal.Field(idx).Int())
	case reflect.Bool:
		adrField.SetBool(newVal.Field(idx).Bool())
	default:
		return
	}
}

// adjustConfigProperties adjusts te config properties for pid, log file, upload dir paths; certs and db files.
func (c *Config) adjustConfigProperties(ctx context.Context, uid int, deployment string) {
	c.Deployment = deployment

	// convert the possible ~/ in Manager_dir to abs path to user's home
	c.ManagerDir = TildaToHome(c.ManagerDir)
	c.ManagerDir += "_" + deployment

	c.convRelativeToAbsManagerPaths()
	c.convRelativeToAbsManagerPathForCert()
	c.convRelativeToAbsManagerPathForDBFiles()
	c.setManagerPort(ctx, uid)
}

// convRelativeToAbsManagerPath converts the possible relative paths of pid, logfile and upload dir to
// abs paths in ManagerDir.
func (c *Config) convRelativeToAbsManagerPaths() {
	if !filepath.IsAbs(c.ManagerPidFile) {
		c.ManagerPidFile = filepath.Join(c.ManagerDir, c.ManagerPidFile)
	}

	if !filepath.IsAbs(c.ManagerLogFile) {
		c.ManagerLogFile = filepath.Join(c.ManagerDir, c.ManagerLogFile)
	}

	if !filepath.IsAbs(c.ManagerUploadDir) {
		c.ManagerUploadDir = filepath.Join(c.ManagerDir, c.ManagerUploadDir)
	}
}

// convRelativeToAbsManagerPathForCert converts the possible relative paths in cert files to
// abs paths in ManagerDir.
func (c *Config) convRelativeToAbsManagerPathForCert() {
	if !filepath.IsAbs(c.ManagerCAFile) {
		c.ManagerCAFile = filepath.Join(c.ManagerDir, c.ManagerCAFile)
	}

	if !filepath.IsAbs(c.ManagerCertFile) {
		c.ManagerCertFile = filepath.Join(c.ManagerDir, c.ManagerCertFile)
	}

	if !filepath.IsAbs(c.ManagerKeyFile) {
		c.ManagerKeyFile = filepath.Join(c.ManagerDir, c.ManagerKeyFile)
	}

	if !filepath.IsAbs(c.ManagerTokenFile) {
		c.ManagerTokenFile = filepath.Join(c.ManagerDir, c.ManagerTokenFile)
	}
}

// convRelativeToAbsManagerPathForDBFiles converts the possible relative paths in db files to
// abs paths in ManagerDir.
func (c *Config) convRelativeToAbsManagerPathForDBFiles() {
	if !filepath.IsAbs(c.ManagerDBFile) {
		c.ManagerDBFile = filepath.Join(c.ManagerDir, c.ManagerDBFile)
	}

	if !filepath.IsAbs(c.ManagerDBBkFile) {
		if !IsRemote(c.ManagerDBBkFile) {
			c.ManagerDBBkFile = filepath.Join(c.ManagerDir, c.ManagerDBBkFile)
		}
	}
}

// setManagerPort sets the cli and web interface ports for manager.
func (c *Config) setManagerPort(ctx context.Context, uid int) {
	// if not explicitly set, calculate ports that no one else would be
	// assigned by us (and hope no other software is using it...)
	if c.ManagerPort == "" {
		c.ManagerPort = calculatePort(ctx, uid, c.Deployment, "cli")
	}

	if c.ManagerWeb == "" {
		c.ManagerWeb = calculatePort(ctx, uid, c.Deployment, "webi")
	}
}

// Calculate a port number that will be unique to this user, deployment and
// ptype ("cli" or "webi").
func calculatePort(ctx context.Context, uid int, deployment string, ptype string) string {
	// get the minimum port number
	pn := getMinPort(uid)

	if pn+3 > maxPort {
		errStr := "Could not calculate a suitable unique port number for you, since your user id is so large;"
		errStr += "please manually set your manager_port and manager_web config options."
		clog.Fatal(ctx, errStr)
	}

	if deployment == Development {
		pn += 2
	}

	if ptype == "webi" {
		pn++
	}

	// it's easier for the things that use this port number if it's a string
	// (because it's used as part of a connection string)
	return strconv.Itoa(pn)
}

// getMinPort calculates and returns the minimum port available.
func getMinPort(uid int) int {
	// our port must be greater than 1024, and by basing on user id we can
	// avoid conflicts with other users of wr on the same machine; we
	// multiply by 4 because we have to reserve 4 ports for each user
	return minPort + (uid * portsPerUser)
}

// DefaultConfig works out the default config for when we need to be able to
// report the default before we know what deployment the user has actually
// chosen, ie. before we have a final config.
func DefaultConfig(ctx context.Context) *Config {
	return ConfigLoadFromNonParentDir(ctx, DefaultDeployment(ctx))
}

// DefaultServer works out the default server (we need this to be able to report
// this default before we know what deployment the user has actually chosen, ie.
// before we have a final config).
func DefaultServer(ctx context.Context) string {
	config := DefaultConfig(ctx)

	return config.ManagerHost + ":" + config.ManagerPort
}

// InS3 tells you if a path is to a file in S3.
func InS3(path string) bool {
	return strings.HasPrefix(path, S3Prefix)
}

// IsRemote tells you if a path is to a remote file system or object store,
// based on its URI.
func IsRemote(path string) bool {
	// (right now we only support S3, but IsRemote is to future-proof us and
	// avoid calling InS3() directly)
	return InS3(path)
}
