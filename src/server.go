package main

import (
	"errors"
	"fmt"
	"github.com/ankur-anand/simple-go-rpc/src/server"
	"github.com/fsnotify/fsnotify"
	ps "github.com/keybase/go-ps"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

var exitServer chan bool
var nvwaSeverConfig *viper.Viper
var nvwaRestoreConfig *viper.Viper

func overrideSystemctl(service string) {
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	criuExe := nvwaSeverConfig.GetString("criu_exe")
	systemdEtc := nvwaSeverConfig.GetString("systemd_etc")

	systemdDir := path.Join(systemdEtc, service+".service.d")
	_ = os.Mkdir(systemdDir, 0700)

	f, err := os.Create(path.Join(systemdDir, "nvwa_override.conf"))
	if err != nil {
		log.Errorf("Unable to create file for %s, err is %s \n", service, err)
		return
	}
	defer f.Close()

	_, err = f.WriteString("[Service]\nExecStart=\nExecStart=")
	if err != nil {
		log.Errorf("Unable to write file for %s, err is %s \n", service, err)
		return
	}

	_, err = f.WriteString(criuExe + " restore" + " -D " + path.Join(criuDir, service) + " -o restore.log --tcp-established --ext-unix-sk --shell-job --daemon -vv")
	if err != nil {
		log.Errorf("Unable to write file for %s, err is %s \n", service, err)
		return
	}
}

func overrideServiceConfig() {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		overrideSystemctl(val)
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

	// dump process memory
	for key, value := range criuPids {
		if value == -1 {
			log.Errorf("Unable to find pid for " + key)
			return 0, errors.New("Unable to find pid for " + key)
		}
		dirPath := path.Join(criuDir, key)
		_ = os.Mkdir(dirPath, 0700)
		wg.Add(1)
		total ++
		go waitCmd(criuExe, []string{"dump", "-D", dirPath,
			"-t", strconv.Itoa(value), "-o", "dump.log", "--tcp-established", "--ext-unix-sk",
			"--shell-job", "--daemon", "-j", "-vv"}, &wg, nil, nil, nil, &success)
	}
	wg.Wait()
	log.Debugf("%d:%d process(es) dump successfully\n", success, total)

	if success < total {
		return 0, errors.New("Some processes dump failed. \n")
	}

	configDir := path.Join(criuDir, "config")
	_ = os.Mkdir(configDir, 0700)

	overrideServiceConfig()
	DumpAllNet(configDir)

	// update kexec image
	err, _ := runCmd(kexecExe, []string{"-l", "/boot/vmlinuz-" + ver,
		"--initrd", "/boot/initramfs-" + ver + ".img"}, nil, nil, nil)
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

func RestoreProcess(ver string) (int, error) {
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
		total ++
		go waitCmd(criuExe, []string{"restore", "-D", path,
			"-o", "restore.log", "--tcp-established", "--ext-unix-sk", "--shell-job",
			"--daemon", "-j", "-vv"}, &wg, os.Stdin, os.Stdout, os.Stderr, &success)
	}
	log.Debugf("Wait criu runs finished \n")
	wg.Wait()
	log.Debugf("%d:%d process(es) restore suceessfully. \n", success, total)
	return 0, nil
}

func runServer(ip, port string) {
	addr := ip + ":" + port
	srv := server.NewServer(addr)
	srv.Register("update", UpdateImage)
	srv.Register("restore", RestoreProcess)
	srv.Register("exit", ExitServer)
	srv.Register("echo", EchoMsg)
	go srv.Run()
}

func startServer(ip, port string, mode int) {
	log.SetLevel(log.DebugLevel)
	exitServer = make(chan bool)

	loadConfig()
	runServer(ip, port)
	if mode == 2 {
		go RestoreProcess("")
	}
	log.Debugf("Server is running in ip %s with port %s \n", ip, port)
	<-exitServer
}
