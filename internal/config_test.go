/*******************************************************************************
 * Copyright (c) 2020 Genome Research Ltd.
 *
 * Author: Ashwini Chhipa <ac55@sanger.ac.uk>
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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-ssg/wr/clog"
)

func fileTestSetup(dir, mport, mweb1, mweb2 string) (string, string, error) {
	path := filepath.Join(dir, ".wr_config.yml")

	_, err := os.Stat(path)
	if err == nil {
		return path, "", &FileExistsError{Path: path, Err: nil}
	}

	path2 := filepath.Join(dir, ".wr_config.development.yml")

	_, err = os.Stat(path2)
	if err == nil {
		return path, "", &FileExistsError{Path: path, Err: nil}
	}

	file, err := os.Create(path)
	if err != nil {
		return path, path2, err
	}

	file2, err := os.Create(path2)
	if err != nil {
		return path, path2, err
	}

	_, err = file.WriteString(fmt.Sprintf("managerport: \"%s\"\n", mport))
	So(err, ShouldBeNil)

	_, err = file.WriteString(fmt.Sprintf("managerweb: \"%s\"\n", mweb1))
	So(err, ShouldBeNil)
	file.Close()

	_, err = file2.WriteString(fmt.Sprintf("managerweb: \"%s\"\n", mweb2))
	So(err, ShouldBeNil)
	file2.Close()

	return path, path2, nil
}

func fileTestTeardown(path, path2 string) {
	err := os.Remove(path)
	if err != nil {
		fmt.Printf("\nfailed to delete %s: %s\n", path, err)
	}

	err = os.Remove(path2)
	if err != nil {
		fmt.Printf("\nfailed to delete %s: %s\n", path2, err)
	}
}

func TestConfig(t *testing.T) {
	ctx := context.Background()

	Convey("Given a path it can check if", t, func() {
		pathS3 := "s3://test1"
		pathNotS3 := "/tmp/test2"

		Convey("it is a path to a file in S3 bucket", func() {
			So(InS3(pathS3), ShouldEqual, true)
			So(InS3(pathNotS3), ShouldEqual, false)
		})

		Convey("it is a remote file system path", func() {
			So(IsRemote(pathS3), ShouldEqual, true)
			So(IsRemote(pathNotS3), ShouldEqual, false)
		})
	})

	Convey("Given a user id", t, func() {
		uid := 1000
		Convey("it can get the minimum port number for it", func() {
			So(getMinPort(uid), ShouldEqual, 5021)
		})

		Convey("it can calculate a unique port for the user", func() {
			Convey("for different deployments and port types", func() {
				So(calculatePort(ctx, uid, "development", "webi"), ShouldEqual, "5024")
				So(calculatePort(ctx, uid, "development", "cli"), ShouldEqual, "5023")
				So(calculatePort(ctx, uid, "production", "webi"), ShouldEqual, "5022")
				So(calculatePort(ctx, uid, "production", "cli"), ShouldEqual, "5021")
			})

			Convey("but not if uid is a big number", func() {
				uid = 65534
				buff := clog.ToBufferAtLevel("crit")
				defer clog.ToDefault()
				os.Setenv("FATAL_EXIT_TEST", "1")
				defer os.Unsetenv("FATAL_EXIT_TEST")
				_ = calculatePort(ctx, uid, "development", "webi")
				bufferStr := buff.String()
				So(bufferStr, ShouldContainSubstring, "fatal=true")
				So(bufferStr, ShouldNotContainSubstring, "caller=clog")
				So(bufferStr, ShouldContainSubstring, "stack=")
				So(bufferStr, ShouldContainSubstring, "user id is so large")
			})
		})
	})

	Convey("Set WR Manager umask", t, func() {
		Convey("When env variable is not set", func() {
			setenvManagerUmask()
			So(os.Getenv("WR_MANAGERUMASK"), ShouldBeEmpty)
		})

		Convey("When env variable is set but umask doesn't have 0 prefix", func() {
			os.Setenv("WR_MANAGERUMASK", "666")
			defer func() {
				os.Unsetenv("WR_MANAGERUMASK")
			}()
			setenvManagerUmask()
			So(os.Getenv("WR_MANAGERUMASK"), ShouldEqual, "666")
		})

		Convey("When env variable is set but umask has 0 prefix", func() {
			os.Setenv("WR_MANAGERUMASK", "0666")
			defer func() {
				os.Unsetenv("WR_MANAGERUMASK")
			}()
			setenvManagerUmask()
			So(os.Getenv("WR_MANAGERUMASK"), ShouldEqual, "666")
		})
	})

	Convey("Given a default wr config", t, func() {
		defConfig := loadDefaultConfig(ctx)
		So(defConfig, ShouldNotBeNil)
		So(defConfig.ManagerPort, ShouldBeEmpty)
		So(defConfig.Source("ManagerPort"), ShouldEqual, "default")
		So(defConfig.ManagerWeb, ShouldBeEmpty)

		Convey("it can check if deployment is production", func() {
			So(defConfig.IsProduction(), ShouldBeTrue)
		})

		Convey("it can clone it", func() {
			clonedConfig := defConfig.clone()
			So(defConfig.ManagerHost, ShouldEqual, clonedConfig.ManagerHost)
			So(defConfig.CloudCIDR, ShouldEqual, clonedConfig.CloudCIDR)
			So(defConfig.CloudDNS, ShouldEqual, clonedConfig.CloudDNS)
			So(defConfig.CloudRAM, ShouldEqual, clonedConfig.CloudRAM)
			So(defConfig.CloudAutoConfirmDead, ShouldEqual, clonedConfig.CloudAutoConfirmDead)
		})

		Convey("and a user id, it can set the manager port", func() {
			uid := 1000
			So(defConfig.ManagerPort, ShouldBeEmpty)
			So(defConfig.ManagerWeb, ShouldBeEmpty)

			defConfig.setManagerPort(ctx, uid)

			So(defConfig.ManagerPort, ShouldEqual, "5021")
			So(defConfig.ManagerWeb, ShouldEqual, "5022")
		})

		Convey("it can convert the relative to Abs path for DB files", func() {
			So(defConfig.ManagerDir, ShouldEqual, "~/.wr")
			So(defConfig.ManagerDBFile, ShouldEqual, "db")
			So(defConfig.ManagerDBBkFile, ShouldEqual, "db_bk")

			defConfig.convRelativeToAbsManagerPathForDBFiles()

			So(defConfig.ManagerDBFile, ShouldEqual, "~/.wr/db")
			So(defConfig.ManagerDBBkFile, ShouldEqual, "~/.wr/db_bk")
		})

		Convey("it can convert the relative to Abs path for certificate files", func() {
			So(defConfig.ManagerDir, ShouldEqual, "~/.wr")
			So(defConfig.ManagerCAFile, ShouldEqual, "ca.pem")
			So(defConfig.ManagerCertFile, ShouldEqual, "cert.pem")
			So(defConfig.ManagerKeyFile, ShouldEqual, "key.pem")
			So(defConfig.ManagerTokenFile, ShouldEqual, "client.token")

			defConfig.convRelativeToAbsManagerPathForCert()

			So(defConfig.ManagerCAFile, ShouldEqual, "~/.wr/ca.pem")
			So(defConfig.ManagerCertFile, ShouldEqual, "~/.wr/cert.pem")
			So(defConfig.ManagerKeyFile, ShouldEqual, "~/.wr/key.pem")
			So(defConfig.ManagerTokenFile, ShouldEqual, "~/.wr/client.token")
		})

		Convey("it can convert the relative to Abs path for other paths", func() {
			So(defConfig.ManagerDir, ShouldEqual, "~/.wr")
			So(defConfig.ManagerPidFile, ShouldEqual, "pid")
			So(defConfig.ManagerLogFile, ShouldEqual, "log")
			So(defConfig.ManagerUploadDir, ShouldEqual, "uploads")

			defConfig.convRelativeToAbsManagerPaths()

			So(defConfig.ManagerPidFile, ShouldEqual, "~/.wr/pid")
			So(defConfig.ManagerLogFile, ShouldEqual, "~/.wr/log")
			So(defConfig.ManagerUploadDir, ShouldEqual, "~/.wr/uploads")
		})

		Convey("user id and deployment type, it can adjust config properties", func() {
			uid := 1000
			deployment := "development"
			userHomeDir, err := os.UserHomeDir()
			So(err, ShouldBeNil)
			expectedManageDir := filepath.Join(userHomeDir, ".wr_"+deployment)

			So(defConfig.ManagerDir, ShouldEqual, "~/.wr")

			defConfig.adjustConfigProperties(ctx, uid, deployment)

			So(defConfig.ManagerDir, ShouldEqual, expectedManageDir)
			So(defConfig.ManagerPidFile, ShouldEqual, filepath.Join(expectedManageDir, "pid"))
			So(defConfig.ManagerCAFile, ShouldEqual, filepath.Join(expectedManageDir, "ca.pem"))
			So(defConfig.ManagerDBFile, ShouldEqual, filepath.Join(expectedManageDir, "db"))
			So(defConfig.ManagerPort, ShouldEqual, "5023")
		})

		Convey("it can also merge with another config", func() {
			otherConfig := loadDefaultConfig(ctx)
			otherConfig.ManagerPort = "2000"

			defConfig.merge(otherConfig, "default")
			So(otherConfig.ManagerPort, ShouldEqual, "2000")
			So(defConfig.ManagerPort, ShouldEqual, "2000")
		})

		Convey("It can be overridden with a config file given its path", func() {
			dir, err := ioutil.TempDir("", "wr_conf_test")
			So(err, ShouldBeNil)
			defer os.RemoveAll(dir)

			mport := "1234"
			mweb1 := "1235"
			mweb2 := "1236"
			path, path2, err := fileTestSetup(dir, mport, mweb1, mweb2)
			defer fileTestTeardown(path, path2)
			So(err, ShouldBeNil)

			defConfig.configLoadFromFile(ctx, path)
			So(defConfig.ManagerPort, ShouldEqual, mport)
			So(defConfig.Source("ManagerPort"), ShouldEqual, path)

			So(defConfig.ManagerWeb, ShouldEqual, mweb1)
			So(defConfig.Source("ManagerWeb"), ShouldEqual, path)

			defConfig.configLoadFromFile(ctx, path2)
			So(defConfig.ManagerPort, ShouldEqual, mport)
			So(defConfig.Source("ManagerPort"), ShouldEqual, path)

			So(defConfig.ManagerWeb, ShouldEqual, mweb2)
			So(defConfig.Source("ManagerWeb"), ShouldEqual, path2)

			_, _, err = fileTestSetup(dir, mport, mweb1, mweb2)
			So(err, ShouldNotBeNil)
		})

		Convey("These can be overridden with config files in WR_CONFIG_DIR", func() {
			uid := 1000
			dir, err := ioutil.TempDir("", "wr_conf_test")
			So(err, ShouldBeNil)
			defer os.RemoveAll(dir)

			mport := "1234"
			mweb1 := "1235"
			mweb2 := "1236"
			path, path2, err := fileTestSetup(dir, mport, mweb1, mweb2)
			defer fileTestTeardown(path, path2)
			So(err, ShouldBeNil)

			os.Setenv("WR_CONFIG_DIR", dir)
			defer func() {
				os.Unsetenv("WR_CONFIG_DIR")
			}()

			defConfig.mergeAllConfigFiles(ctx, uid, "production", "", false)

			So(defConfig.ManagerPort, ShouldEqual, mport)
			So(defConfig.Source("ManagerPort"), ShouldEqual, path)
			So(defConfig.ManagerWeb, ShouldEqual, mweb1)
			So(defConfig.Source("ManagerWeb"), ShouldEqual, path)

			Convey("These can be overridden with config files in home dir", func() {
				realHome, err := os.UserHomeDir()
				So(err, ShouldBeNil)
				newHome, err := ioutil.TempDir(dir, "home")
				So(err, ShouldBeNil)
				os.Setenv("HOME", newHome)
				defer func() {
					os.Setenv("HOME", realHome)
				}()
				home, err := os.UserHomeDir()
				So(err, ShouldBeNil)
				So(home, ShouldNotEqual, realHome)

				mport := "1334"
				mweb1 := "1335"
				mweb2 := "1336"
				path3, path4, err := fileTestSetup(home, mport, mweb1, mweb2)
				defer fileTestTeardown(path3, path4)
				So(err, ShouldBeNil)

				defConfig.mergeAllConfigFiles(ctx, uid, "production", "", true)
				So(defConfig.ManagerPort, ShouldEqual, mport)
				So(defConfig.Source("ManagerPort"), ShouldEqual, path3)
				So(defConfig.ManagerWeb, ShouldEqual, mweb1)
				So(defConfig.Source("ManagerWeb"), ShouldEqual, path3)

				Convey("not if home directory is empty", func() {
					os.Unsetenv("HOME")
					buff := clog.ToBufferAtLevel("fatal")
					defer clog.ToDefault()
					os.Setenv("FATAL_EXIT_TEST", "1")
					defer func() {
						os.Setenv("HOME", realHome)
						os.Unsetenv("FATAL_EXIT_TEST")
					}()
					defConfig.mergeAllConfigFiles(ctx, uid, "production", "", true)

					bufferStr := buff.String()
					So(bufferStr, ShouldContainSubstring, "fatal=true")
					So(bufferStr, ShouldNotContainSubstring, "caller=clog")
					So(bufferStr, ShouldContainSubstring, "stack=")
				})

				Convey("These can be overridden with config files in current dir", func() {
					pwd, err := os.Getwd()
					So(err, ShouldBeNil)
					mport = "1434"
					mweb1 = "1435"
					mweb2 = "1436"
					path5, path6, err := fileTestSetup(pwd, mport, mweb1, mweb2)
					defer fileTestTeardown(path5, path6)
					So(err, ShouldBeNil)

					defConfig.mergeAllConfigFiles(ctx, uid, "production", pwd, true)
					So(defConfig.ManagerPort, ShouldEqual, mport)
					So(defConfig.Source("ManagerPort"), ShouldEqual, path5)
					So(defConfig.ManagerWeb, ShouldEqual, mweb1)
					So(defConfig.Source("ManagerWeb"), ShouldEqual, path5)
				})
			})
		})
	})

	Convey("Set source on the change of a config field property", t, func() {
		type testConfig struct {
			ManagerHost        string  `default:"localhost"`
			ManagerUmask       int     `default:"7"`
			RandomFloatValue   float32 `default:"1.1"`
			ManagerSetDomainIP bool    `default:"false"`
		}

		old := &testConfig{}
		old.ManagerUmask = 4

		new := &testConfig{}

		v := reflect.ValueOf(*old)
		typeOfC := v.Type()

		adrFieldString := reflect.ValueOf(new).Elem().Field(0)
		adrFieldBool := reflect.ValueOf(new).Elem().Field(1)
		adrFieldInt := reflect.ValueOf(new).Elem().Field(2)
		adrFieldFloat := reflect.ValueOf(new).Elem().Field(3)

		setSourceOnChangeProp(typeOfC, adrFieldString, v, 0)
		setSourceOnChangeProp(typeOfC, adrFieldBool, v, 1)
		setSourceOnChangeProp(typeOfC, adrFieldInt, v, 2)
		setSourceOnChangeProp(typeOfC, adrFieldFloat, v, 3)

		So(new.ManagerUmask, ShouldEqual, 4)
	})

	Convey("It can get the default deployment", t, func() {
		Convey("when it not running in server mode", func() {
			defDeployment := DefaultDeployment(ctx)
			So(defDeployment, ShouldEqual, "production")

			Convey("it can get overridden if WR_DEPLOYMENT env variable is set", func() {
				os.Setenv("WR_DEPLOYMENT", "development")
				defer func() {
					os.Unsetenv("WR_DEPLOYMENT")
				}()

				defDeployment := DefaultDeployment(ctx)
				So(defDeployment, ShouldEqual, "development")
			})
		})

		Convey("when it is running in server mode", func() {
			orgPWD, err := os.Getwd()
			So(err, ShouldBeNil)

			dir, err := ioutil.TempDir("", "wr_conf_test")
			So(err, ShouldBeNil)
			path := dir + "/jobqueue/server.go"
			err = os.MkdirAll(path, 0777)
			So(err, ShouldBeNil)
			defer func() {
				defer os.RemoveAll(dir)
			}()

			err = os.Chdir(dir)
			So(err, ShouldBeNil)
			defer func() {
				err = os.Chdir(orgPWD)
			}()

			defDeployment := DefaultDeployment(ctx)
			So(defDeployment, ShouldEqual, "development")
		})
	})

	Convey("It can create a config with env vars", t, func() {
		os.Setenv("WR_MANAGERPORT", "1234")
		os.Setenv("WR_MANAGERUMASK", "77")
		os.Setenv("WR_MANAGERSETDOMAINIP", "true")
		defer func() {
			os.Unsetenv("WR_MANAGERPORT")
			os.Unsetenv("WR_MANAGERUMASK")
			os.Unsetenv("WR_MANAGERSETDOMAINIP")
		}()

		envVarConfig := getEnvVarsConfig(ctx)

		So(envVarConfig.ManagerPort, ShouldEqual, "1234")
		So(envVarConfig.ManagerWeb, ShouldBeEmpty)
		So(envVarConfig.ManagerUmask, ShouldEqual, 77)
		So(envVarConfig.ManagerSetDomainIP, ShouldBeTrue)
	})

	Convey("It can merge Default Config and Env Var config", t, func() {
		os.Setenv("WR_MANAGERPORT", "1234")
		os.Setenv("WR_MANAGERUMASK", "077")
		os.Setenv("WR_MANAGERSETDOMAINIP", "true")
		defer func() {
			os.Unsetenv("WR_MANAGERPORT")
			os.Unsetenv("WR_MANAGERUMASK")
			os.Unsetenv("WR_MANAGERSETDOMAINIP")
		}()

		mergedConfig := mergeDefaultAndEnvVarsConfigs(ctx)
		So(mergedConfig.ManagerPort, ShouldEqual, "1234")
		So(mergedConfig.Source("ManagerPort"), ShouldEqual, ConfigSourceEnvVar)
		So(mergedConfig.ManagerWeb, ShouldBeEmpty)
		So(mergedConfig.Source("ManagerWeb"), ShouldEqual, ConfigSourceDefault)
		So(mergedConfig.ManagerUmask, ShouldEqual, 77)
		So(mergedConfig.Source("ManagerUmask"), ShouldEqual, ConfigSourceEnvVar)
		So(mergedConfig.ManagerSetDomainIP, ShouldBeTrue)
		So(mergedConfig.Source("ManagerSetDomainIP"), ShouldEqual, ConfigSourceEnvVar)
	})

	Convey("It can merge all the configs and return a final config", t, func() {
		// env config
		os.Setenv("WR_MANAGERUMASK", "077")
		defer os.Unsetenv("WR_MANAGERUMASK")

		// config file in pwd
		uid := 1000
		pwd, err := os.Getwd()
		So(err, ShouldBeNil)
		mport := "5434"
		mweb1 := "5435"
		mweb2 := "5436"
		path7, path8, err := fileTestSetup(pwd, mport, mweb1, mweb2)
		defer fileTestTeardown(path7, path8)
		So(err, ShouldBeNil)

		finalConfig := mergeAllConfigs(ctx, uid, "production", pwd, true)
		So(finalConfig.ManagerPort, ShouldEqual, mport)
		So(finalConfig.Source("ManagerPort"), ShouldEqual, path7)
		So(finalConfig.ManagerWeb, ShouldEqual, mweb1)
		So(finalConfig.Source("ManagerWeb"), ShouldEqual, path7)
		So(finalConfig.ManagerUmask, ShouldEqual, 77)
		So(finalConfig.Source("ManagerUmask"), ShouldEqual, ConfigSourceEnvVar)

		Convey("it can override deployment to default deployment if it's not development or production", func() {
			finalConfig := mergeAllConfigs(ctx, uid, "testDeployment", pwd, true)
			So(finalConfig.IsProduction(), ShouldBeTrue)
		})
	})

	Convey("ConfigLoad* gives default values to start with", t, func() {
		if os.Getuid() > 16128 {
			fmt.Print(os.Getuid())
			fmt.Printf("Failed to calculate a suitable unique port number since your user id is so large.\n")
			fmt.Printf("Skipping tests...\n")
			t.Skip("skipping tests; user id is very large.")
		}

		Convey("When loaded from parent directory", func() {
			config := ConfigLoadFromParentDir(ctx, "development")
			So(config, ShouldNotBeNil)
			So(config.IsProduction(), ShouldBeFalse)
		})

		Convey("When loaded from non parent directory", func() {
			config := ConfigLoadFromNonParentDir(ctx, "testing")
			So(config, ShouldNotBeNil)
			So(config.IsProduction(), ShouldBeTrue)
		})

		Convey("It can get the default config", func() {
			config := DefaultConfig(ctx)
			So(config, ShouldNotBeNil)
			So(config.IsProduction(), ShouldBeTrue)

			Convey("It can get the Default server", func() {
				server := DefaultServer(ctx)
				So(server, ShouldEqual, config.ManagerHost+":"+config.ManagerPort)
			})
		})
	})
}
