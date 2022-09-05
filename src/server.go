package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type rpmFunc func(string) int

var exitServer chan os.Signal
var rpmList = map[string]rpmFunc{}
var nvwaSeverConfig *viper.Viper
var nvwaRestoreConfig *viper.Viper

func registerRPC(cmd string, rpc rpmFunc) {
	if v, e := rpmList[cmd]; e {
		log.Errorf("%s exist with func %s \n", cmd,
			runtime.FuncForPC(reflect.ValueOf(v).Pointer()).Name())
		return
	}
	rpmList[cmd] = rpc
}

func overrideConf(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, err is %s \n", path, err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Errorf("Unable to write file %s, err is %s \n", path, err)
		return err
	}
	return nil
}

func getSystemdOptions(service string, option string) (error, string) {
	return runCmd("systemctl", []string{"show", "--property",
		option, "--value", service}, nil, nil, nil)
}

func getPIDFile(service string) (error, string) {
	err, ret := getSystemdOptions(service, "PIDFile")
	if err != nil {
		log.Errorf("Unable to get pid file for service %s\n", service)
		log.Errorf("Error is %s \n", err)
		return err, ""
	}

	log.Debugf("Get pid file for %s - %s \n", service, ret)

	i := strings.Index(ret, "=")
	return nil, ret[i+1:]
}

func overrideSystemctl(service string, pid int) error {
	systemdEtc := nvwaSeverConfig.GetString("systemd_etc")

	systemdDir := path.Join(systemdEtc, service+".service.d")
	_ = os.Mkdir(systemdDir, 0700)

	content := "[Service]\nExecStart=\nExecStart="
	content += "nvwa restore " + service + "@" + strconv.Itoa(pid) + "\n"
	content += "User=root\nGroup=root\n"

	err := overrideConf(path.Join(systemdDir, "nvwa_override_exec.conf"), content)
	if err != nil {
		return err
	}

	content = "[Unit]\nAfter=nvwa.service network-online.target\n"
	content += "[Service]\nRestart=no\n"
	err = overrideConf(path.Join(systemdDir, "nvwa_override_restart.conf"), content)
	if err != nil {
		return err
	}
	return nil
}

func overrideServiceConfig(criuPids map[string]int) error {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		if err := overrideSystemctl(val, criuPids[val]); err != nil {
			return err
		}
	}
	return nil
}

func getPidName(pid string) (string, error) {
	data, err := ioutil.ReadFile("/proc/" + pid + "/cmdline")
	if err != nil {
		log.Errorf("Unable to find name for pid %s \n", pid)
		log.Errorf("Error is %s \n", err)
		return "", err
	}
	names := strings.Split(strings.TrimSuffix(string(data), "\n"), "/")
	name := names[len(names)-1]
	if len(name) == 0 {
		name = pid
	}
	name = strings.Replace(name, "\x00", "", -1)
	log.Debugf("Find name %s for pid %s", name, pid)
	return name, nil
}

func findPids(criuPids map[string]int) error {
	pidNames := nvwaRestoreConfig.GetStringSlice("pids")
	for _, val := range pidNames {
		pid, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Unable to get pid from %s \n", val)
			log.Errorf("Error is %s \n", err)
			return err
		}
		_, err = getPidName(val)
		if err != nil {
			return err
		}
		criuPids[val] = pid
	}

	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		err, tmpRet := getSystemdOptions(val, "MainPID")
		if err != nil {
			log.Errorf("Unable to get pid for service %s\n", val)
			log.Errorf("Error is %s \n", err)
			return err
		}
		ret, err := strconv.Atoi(strings.TrimSpace(tmpRet))
		if err != nil {
			if ret == 0 {
				err = fmt.Errorf("Unable to get pid for service %s, error is %s, ret is %d \n",
					val, err, ret)
			}
			log.Errorf("%s \n", err)
			return err
		}
		criuPids[val] = ret
		log.Debugf("Get pid %d for service %s \n", criuPids[val], val)
	}
	return nil
}

func removeCriuImage(psName string) error {
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	err := os.RemoveAll(path.Join(criuDir, psName))
	if err != nil {
		log.Errorf("Unable to remove dump dirctory for %s err %s \n", psName, err)
		return err
	}
	return nil
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

func removeAllOverrideSys() {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		removeOverrideSystemctl(val)
	}
}

func removeServiceImg() {
	services := nvwaRestoreConfig.GetStringSlice("services")
	for _, val := range services {
		removeCriuImage(val)
	}
}

func removeProcessImg() {
	pidNames := nvwaRestoreConfig.GetStringSlice("pids")
	for _, val := range pidNames {
		removeCriuImage(val)
	}
}

func InitEnv(env string) int {
	log.Debugf("Init Env \n")
	enablePM := nvwaRestoreConfig.GetBool("enable_pin_memory")

	removeAllOverrideSys()
	removeProcessImg()
	removeServiceImg()
	if enablePM {
		pinMemoryClear()
	}
	return 0
}

func loadCmdline() (string, error) {
	data, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Errorf("Unable to read cmdline, error is %s \n", err)
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), err
}

func getCriuExtPara() []string {
	criuExtPara := []string{"--shell-job", "--enable-external-masters", "--skip-in-flight"}

	enablePM := nvwaRestoreConfig.GetBool("enable_pin_memory")
	if enablePM {
		criuExtPara = append(criuExtPara, "--pin-memory")
	}

	return criuExtPara
}

func getCriuPara(op string, dir string, pid string) []string {
	criuBasicPara := []string{}
	criuExtPara := getCriuExtPara()
	if op == "dump" {
		criuBasicPara = []string{op, "-D", dir, "-t", pid,
			"-o", "dump.log", "--tcp-established", "--ext-unix-sk",
			"--file-locks", "--daemon", "-vv"}
	} else if op == "restore" {
		criuBasicPara = []string{op, "-D", dir,
			"-o", "restore.log", "--tcp-established", "--ext-unix-sk",
			"--file-locks", "--daemon", "-vv"}
	}
	return append(criuBasicPara, criuExtPara...)
}

func UpdateImage(ver string) int {
	var wg sync.WaitGroup
	total := 0
	success := 0
	criuPids := make(map[string]int)
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	criuExe := nvwaSeverConfig.GetString("criu_exe")
	kexecExe := nvwaSeverConfig.GetString("kexec_exe")
	enableQK := nvwaRestoreConfig.GetBool("enable_quick_kexec")
	enablePM := nvwaRestoreConfig.GetBool("enable_pin_memory")

	if criuDir == "" {
		log.Errorf("Missing criuDir settings in config file \n")
		return -1
	}

	kexecLoad := "-l"
	if enableQK {
		kexecLoad = "-q"
	}

	cmdline, err := loadCmdline()
	if err != nil {
		log.Error(err)
		return -1
	}

	err, _ = runCmd(kexecExe, []string{kexecLoad, "/boot/vmlinuz-" + ver,
		"--initrd", "/boot/initramfs-" + ver + ".img", "--append=" +
			cmdline}, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		log.Errorf("Unable to load kernel image, err is %s \n", err)
		return -1
	}

	err = findPids(criuPids)
	if err != nil {
		return -1
	}

	err = overrideServiceConfig(criuPids)
	if err != nil {
		return -1
	}

	if enablePM {
		err = pinMemoryClear()
		if err != nil {
			log.Errorf("Execute clear pin memory failed \n")
			log.Errorf("Error: %s \n", err)
			return -1
		}
	}

	for key, value := range criuPids {
		dirPath := path.Join(criuDir, key)
		_ = os.Mkdir(dirPath, 0700)
		wg.Add(1)
		total++
		go waitCmd(criuExe, getCriuPara("dump", dirPath, strconv.Itoa(value)), &wg,
			os.Stdin, os.Stdout, os.Stderr, &success)
		if enablePM {
			/* limitation of pin-memory, only support serial execution */
			wg.Wait()
		}
	}
	wg.Wait()
	log.Debugf("%d:%d process(es) dump successfully\n", success, total)

	if success < total {
		removeAllOverrideSys()
		log.Errorf("Some processes dump failed.\n")
		return -1
	}

	if enablePM {
		err = pinMemoryFinish()
		if err != nil {
			log.Errorf("Execute finish pin memory failed \n")
			log.Errorf("Error: %s \n", err)
			return -1
		}
	}
	configDir := path.Join(criuDir, "config")
	_ = os.Mkdir(configDir, 0700)

	DumpAllNet(configDir)

	err, _ = runCmd(kexecExe, []string{"-e", "-x"}, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		log.Errorf("Unable to run kexec -e with err %s \n", err)
		return -1
	}
	return 0
}

func readConfig(curConfig *viper.Viper, name string) {
	curConfig.SetConfigName(name)
	curConfig.SetConfigType("yaml")
	curConfig.AddConfigPath(".")
	curConfig.AddConfigPath("./config")
	curConfig.AddConfigPath("/etc/nvwa/")
	err := curConfig.ReadInConfig()
	if err != nil {
		log.Fatalf("Load config %s failed, %s \n", name, err)
	}
	curConfig.WatchConfig()
	curConfig.OnConfigChange(func(e fsnotify.Event) {
		log.Debugf("Config file %s changed", e.Name)
	})
}

func loadConfig() {
	nvwaSeverConfig = viper.New()
	nvwaRestoreConfig = viper.New()
	readConfig(nvwaSeverConfig, "nvwa-server")
	readConfig(nvwaRestoreConfig, "nvwa-restore")
}

func RestoreService(cmd string) int {
	i := strings.Index(cmd, "@")
	service := cmd[:i]
	pid := cmd[i+1:]

	log.Debugf("nvwa restore %s %s \n", service, pid)

	criuExe := nvwaSeverConfig.GetString("criu_exe")
	criuDir := nvwaSeverConfig.GetString("criu_dir")

	enbale_debug_mode := nvwaRestoreConfig.GetBool("enbale_debug_mode")
	if enbale_debug_mode {
		log.SetLevel(log.DebugLevel)
	}

	err, pidfile := getPIDFile(service)
	if err != nil {
		return -1
	}
	pidfile = strings.TrimSpace(pidfile)
	if pidfile != "" {
		pidData := []byte(pid)
		_ = ioutil.WriteFile(pidfile, pidData, 0644)
	}

	err, _ = runCmd(criuExe, getCriuPara("restore", path.Join(criuDir, service), ""),
		os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		log.Errorf("Restore %s failed, error is %s \n", service, err)
		return -1
	}
	log.Debugf("Restore service %s successfully \n", service)
	removeOverrideSystemctl(service)
	if !enbale_debug_mode {
		removeCriuImage(service)
	}
	return 0
}

func restoreProcess() {
	var wg sync.WaitGroup
	total := 0
	success := 0
	criuDir := nvwaSeverConfig.GetString("criu_dir")
	criuExe := nvwaSeverConfig.GetString("criu_exe")

	enbale_debug_mode := nvwaRestoreConfig.GetBool("enbale_debug_mode")
	if enbale_debug_mode {
		log.SetLevel(log.DebugLevel)
	}

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
		go waitCmd(criuExe, getCriuPara("restore", path, ""),
			&wg, os.Stdin, os.Stdout, os.Stderr, &success)
	}
	log.Debugf("Wait criu runs finished \n")
	wg.Wait()
	log.Debugf("%d:%d process(es) restore suceessfully. \n", success, total)
	if success < total {
		log.Debugf("Some process(es) restore failed,\n" +
			"check nvwa log and init enviroment before next trial")
	} else if !enbale_debug_mode {
		removeProcessImg()
	}
	return
}

func handleCMD(cmd string) int {
	cmds := strings.Split(cmd, " ")
	if len(cmds) != 3 {
		log.Errorf("Get wrong cmd len %d \n", len(cmds))
		return -1
	}
	if cmds[0] != "nvwa:" {
		log.Errorf("Get wrong secret %s \n", cmds[0])
		return -1
	}
	if v, e := rpmList[cmds[1]]; e {
		return v(cmds[2])
	}
	log.Errorf("%s is not registered \n", cmds[1])
	return -1
}

func ExitServer(msg string) int {
	exitServer <- syscall.SIGINT
	log.Debugf("Server will exit \n")
	return 0
}

func may_init_socket(path string) error {
	socketDir := filepath.Dir(path)
	log.Debugf("Socket directory %s \n", socketDir)
	if _, err := os.Stat(socketDir); os.IsNotExist(err) {
		return os.Mkdir(socketDir, 0700)
	}
	return nil
}

func runServer(path string) {
	registerRPC("update", UpdateImage)
	registerRPC("restore", RestoreService)
	registerRPC("init", InitEnv)
	registerRPC("exit", ExitServer)

	err := may_init_socket(path)
	if err != nil {
		log.Fatal(err)
	}

	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		log.Fatal(err)
	}

	l, err := net.ListenUnix("unix", addr)
	if err != nil {
		log.Errorf("Please ensure run it as root. \n")
		log.Errorf("Ensure no other nvwa process is running.")
		log.Errorf("And remove %s mannually if necessary. \n", path)
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				log.Fatal(err)
			}
			var buf [1024]byte
			n, err := conn.Read(buf[:])
			if err != nil {
				log.Fatal(err)
			}
			ret := handleCMD(string(buf[:n]))
			_, err = conn.Write([]byte(strconv.Itoa(ret)))
			if err != nil {
				log.Fatal(err)
			}
			conn.Close()
		}
	}()
}

func clearServer(socketPath string) {
	os.Remove(socketPath)
}

func startServer(socketPath string) {
	log.SetLevel(log.WarnLevel)
	exitServer = make(chan os.Signal)
	signal.Notify(exitServer, os.Interrupt, syscall.SIGTERM)

	loadConfig()
	runServer(socketPath)
	NotifySytemd()
	go restoreProcess()
	log.Debugf("Server is listening in %s \n", socketPath)
	<-exitServer
	clearServer(socketPath)
}
