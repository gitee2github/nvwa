package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"io/ioutil"

	"github.com/ankur-anand/simple-go-rpc/src/server"
	"github.com/fsnotify/fsnotify"
	ps "github.com/keybase/go-ps"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var exitServer chan bool
var nvwaSeverConfig *viper.Viper
var nvwaRestoreConfig *viper.Viper

func overrideConf(path string, content string) {
	f, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, err is %s \n", path, err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Errorf("Unable to write file %s, err is %s \n", path, err)
		return
	}
}

func overrideSystemctl(service string, pid int) {
	systemdEtc := nvwaSeverConfig.GetString("systemd_etc")

	systemdDir := path.Join(systemdEtc, service+".service.d")
	_ = os.Mkdir(systemdDir, 0700)

	content := "[Service]\nExecStart=\nExecStart="
	content += "nvwa restore " + service + " " + strconv.Itoa(pid) + "\n"
	content += "User=root\nGroup=root\n"
	overrideConf(path.Join(systemdDir, "nvwa_override_exec.conf"), content)

	content = "[Unit]\nAfter=nvwa.service network-online.target\n"
	content += "[Service]\nRestart=no\n"
	overrideConf(path.Join(systemdDir, "nvwa_override_restart.conf"), content)
}

func overrideServiceConfig(criuPids map[string]int) {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		overrideSystemctl(val, criuPids[val])
	}
}

// with same process name, use the minimum pid
func findPids(criuPids map[string]int) {
	pidNames := nvwaRestoreConfig.GetStringSlice("pids")
	for _, val := range pidNames {
		criuPids[val] = -1
	}

	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		criuPids[val] = -1
		err, tmpRet := runCmd("systemctl", []string{"show", "--property",
			"MainPID", "--value", val}, nil, nil, nil)
		if err != nil {
			log.Errorf("Unable to get pid for service %s, err is %s \n", val, err)
			continue
		}
		ret, err := strconv.Atoi(strings.TrimSpace(tmpRet))
		if err != nil || ret == 0 {
			log.Errorf("Unable to get pid for service %s, err is %s, ret is %d \n", val, err, ret)
			continue
		}
		criuPids[val] = ret
		log.Debugf("Get pid %d for service %s \n", criuPids[val], val)
	}

	processList, err := ps.Processes()
	if err != nil {
		log.Errorf("Unable to find processes, err is %s", err)
		return
	}

	for x := range processList {
		process := processList[x]
		pid, ok := criuPids[process.Executable()]
		if ok && (pid == -1 || process.Pid() < pid) {
			criuPids[process.Executable()] = process.Pid()
		}
	}
}

func removeOverrideSystemctl(service string) error {
	systemdEtc := nvwaSeverConfig.GetString("systemd_etc")
	systemdDir := path.Join(systemdEtc, service+".service.d")

	err := os.Remove(path.Join(systemdDir, "nvwa_override_exec.conf"))
	if err != nil {
		log.Errorf("Unable to remove exec service file for %s err %s \n", service, err)
		return err
	}

	err = os.Remove(path.Join(systemdDir, "nvwa_override_restart.conf"))
	if err != nil {
		log.Errorf("Unable to remove restart service file for %s err %s \n", service, err)
		return err
	}

	return nil
}

func removePidImage(psName string) error {
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	err := os.RemoveAll(path.Join(criuDir, psName))
	if err != nil {
		log.Errorf("Unable to remove dump dirctory for %s err %s \n", psName, err)
		return err
	}
	return nil
}

func removeAllOverrideSys() {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		removeOverrideSystemctl(val)
	}
}

func removeAllPids() {
	pidNames := nvwaRestoreConfig.GetStringSlice("pids")
	for _, val := range pidNames {
		removePidImage(val)
	}
}

func InitEnv(env string) (int, error) {
	removeAllOverrideSys()
	removeAllPids()
	return 0, nil
}

func loadCmdline() (string, error) {
	data, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Errorf("Unable to read cmdline, error is %s \n", err)
		return "", err
	}
	return string(data), err
}

func UpdateImage(ver string) (int, error) {
	var wg sync.WaitGroup
	total := 0
	success := 0
	criuPids := make(map[string]int)
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	criuExe := nvwaSeverConfig.GetString("criu_exe")
	kexecExe := nvwaSeverConfig.GetString("kexec_exe")

	if criuDir == "" {
		log.Errorf("Missing criuDir settings in config file \n")
		return 0, errors.New("Missing criuDir settings in config file")
	}

	findPids(criuPids)

	overrideServiceConfig(criuPids)
	// dump process memory
	for key, value := range criuPids {
		if value == -1 {
			log.Errorf("Unable to find pid for " + key)
			return 0, errors.New("Unable to find pid for " + key)
		}
		dirPath := path.Join(criuDir, key)
		_ = os.Mkdir(dirPath, 0700)
		wg.Add(1)
		total++
		go waitCmd(criuExe, []string{"dump", "-D", dirPath,
			"-t", strconv.Itoa(value), "-o", "dump.log", "--tcp-established", "--ext-unix-sk",
			"--shell-job", "--daemon", "-j", "-vv"}, &wg, nil, nil, nil, &success)
	}
	wg.Wait()
	log.Debugf("%d:%d process(es) dump successfully\n", success, total)

	if success < total {
		removeAllOverrideSys()
		return 0, errors.New("Some processes dump failed.\n")
	}

	
	configDir := path.Join(criuDir, "config")
	_ = os.Mkdir(configDir, 0700)

	DumpAllNet(configDir)

	cmdline, err := loadCmdline()
	if err != nil {
		return 0, err
	}

	// update kexec image
	err, _ = runCmd(kexecExe, []string{"-q", "/boot/vmlinuz-" + ver,
		"--initrd", "/boot/initramfs-" + ver + ".img", "--append="+cmdline},
		nil, nil, nil)
	if err != nil {
		log.Errorf("Unable to load kernel image, err is %s \n", err)
		return 0, err
	}

	err, _ = runCmd(kexecExe, []string{"-e"}, nil, nil, nil)
	if err != nil {
		log.Errorf("Unable to run kexec -e with err %s \n", err)
	}
	return 0, nil
}

func ExitServer(msg string) (int, error) {
	exitServer <- true
	log.Debugf("Server will exit \n")
	return 0, nil
}

func EchoMsg(msg string) (int, error) {
	log.Debugf("Get msg " + msg)
	return 0, nil
}

func readConfig(curConfig *viper.Viper, name string) {
	curConfig.SetConfigName(name)
	curConfig.SetConfigType("yaml")
	curConfig.AddConfigPath(".")
	curConfig.AddConfigPath("./config")
	curConfig.AddConfigPath("/etc/nvwa/")
	err := curConfig.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Load config %s failed, %s \n", name, err))
	}
	curConfig.WatchConfig()
	curConfig.OnConfigChange(func(e fsnotify.Event) {
		log.Errorf("Config file changed: please restart the server", e.Name)
	})
}

func loadConfig() {
	nvwaSeverConfig = viper.New()
	nvwaRestoreConfig = viper.New()
	readConfig(nvwaSeverConfig, "nvwa-server")
	readConfig(nvwaRestoreConfig, "nvwa-restore")
}

func RestoreService(service string) (int, error) {
	criuExe := nvwaSeverConfig.GetString("criu_exe")
	criuDir := nvwaSeverConfig.GetString("criu_dir")

	err, _ := runCmd(criuExe, []string{"restore", "-D", path.Join(criuDir, service),
		"-o", "restore.log", "--tcp-established", "--ext-unix-sk",
		"--shell-job", "--daemon", "-j", "-vv"}, nil, nil, nil)
	if (err != nil) {
		log.Errorf("Restore %s failed, error is %s \n", service, err)
		return 0, err
	}
	log.Debugf("Restore service %s successfully \n", service)
	removeOverrideSystemctl(service)
	return 0, nil
}

func RestoreProcess() {
	var wg sync.WaitGroup
	total := 0
	success := 0
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	criuExe := nvwaSeverConfig.GetString("criu_exe")

	enableNet := nvwaRestoreConfig.GetBool("restore_net")
	if enableNet {
		configDir := path.Join(criuDir, "config")
		RestoreAllNet(configDir)
	}

	pidNames := nvwaRestoreConfig.GetStringSlice("pids")
	for _, val := range pidNames {
		path := criuDir + "/" + val
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		wg.Add(1)
		total++
		go waitCmd(criuExe, []string{"restore", "-D", path,
			"-o", "restore.log", "--tcp-established", "--ext-unix-sk", "--shell-job",
			"--daemon", "-j", "-vv"}, &wg, os.Stdin, os.Stdout, os.Stderr, &success)
	}
	log.Debugf("Wait criu runs finished \n")
	wg.Wait()
	log.Debugf("%d:%d process(es) restore suceessfully. \n", success, total)
	return
}

func runServer(ip, port string) {
	addr := ip + ":" + port
	srv := server.NewServer(addr)
	srv.Register("update", UpdateImage)
	srv.Register("restore", RestoreService)
	srv.Register("exit", ExitServer)
	srv.Register("echo", EchoMsg)
	srv.Register("init", InitEnv)
	go srv.Run()
}

func startServer(ip, port string, mode int) {
	var wg sync.WaitGroup
	log.SetLevel(log.DebugLevel)
	exitServer = make(chan bool)

	loadConfig()
	runServer(ip, port)
	NotifySytemd()
	if mode == 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RestoreProcess()
		}()
	}
	wg.Wait()
	log.Debugf("Server is running in ip %s with port %s \n", ip, port)
	<-exitServer
}
