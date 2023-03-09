package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	ProjectName = "GENERAL"
	Username    = "admin"
)

type Paths struct {
	Repo               string `mapstructure:"repo"`
	DBToolPath         string `mapstructure:"DBToolPath"`
	PropertiesFilePath string `mapstructure:"propertiesFilePath"`
	LogsPath           string `mapstructure:"logsPath"`
}

type Probe struct {
	Run       bool   `mapstructure:"run"`
	ProbeName string `mapstructure:"probeName"`
	JarPath   string `mapstructure:"jarPath"`
}

type ImportData struct {
	Run       bool   `mapstructure:"run"`
	ProbeName string `mapstructure:"probeName"`
	ProbeXML  string `mapstructure:"probeXML"`
}

type ImportConfig struct {
	Run  bool   `mapstructure:"run"`
	Path string `mapstructure:"path"`
}

type Actions struct {
	CreateDB      bool           `mapstructure:"createDB"`
	Initialize    bool           `mapstructure:"initialize"`
	ImportConfigs []ImportConfig `mapstructure:"importConfigs"`
	CreateProject bool           `mapstructure:"createProject"`
	ImportData    []ImportData   `mapstructure:"importData"`
	Probes        []Probe        `mapstructure:"probes"`
}

type CROSSConfig struct {
	PrintExecution bool    `mapstructure:"printExecution"`
	Paths          Paths   `mapstructure:"paths"`
	Actions        Actions `mapstructure:"actions"`
}

func (c CROSSConfig) RunActions() {
	c.goToDBToolPath()
	c.createDB()
	c.initialize()
	c.importConfigCommands()
	c.createProject()
	c.importData()
	c.runProbes()
}

func (c CROSSConfig) goToDBToolPath() {
	err := os.Chdir(c.Paths.DBToolPath)
	if err != nil {
		panic(err)
	}
}

func (c CROSSConfig) createDB() {
	if !c.Actions.CreateDB {
		return
	}

	color.HiMagenta("Running createDB command")

	command := defaultDBToolCommand(c, true)
	command = append(command, "createDatabase")
	command = append(command, "--drop-if-exists")

	err := c.executeCommand("java", command, "")
	if err != nil {
		panic(err)
	}
}

func (c CROSSConfig) initialize() {
	if !c.Actions.Initialize {
		return
	}
	color.HiMagenta("Running initialize command")
	command := defaultDBToolCommand(c, true)
	command = append(command, "initialize")

	err := c.executeCommand("java", command, "")
	if err != nil {
		panic(err)
	}
}

func (c CROSSConfig) importConfigCommands() {
	importConfigs := c.Actions.ImportConfigs
	if importConfigs == nil {
		return
	}
	if len(importConfigs) == 0 {
		return
	}

	for i := 0; i < len(importConfigs); i++ {
		commands := defaultDBToolCommand(c, true)
		cmd := importConfigs[i]
		if !cmd.Run {
			continue
		}
		commands = append(commands, "importConfig")
		if strings.Split(cmd.Path, "")[0] != "-" {
			commands = append(commands, "-f")
		}
		commands = append(commands, cmd.Path)

		color.HiMagenta("Running command: " + strings.Join(commands, " "))
		err := c.executeCommand("java", commands, "")
		if err != nil {
			panic(err)
		}
	}
}

func (c CROSSConfig) createProject() {
	if !c.Actions.CreateProject {
		return
	}

	color.HiMagenta("Running createProject command")

	command := defaultDBToolCommand(c, true)
	command = append(command, "createProject")
	command = append(command, "-n="+ProjectName)
	command = append(command, "-t="+ProjectName)

	err := c.executeCommand("java", command, "")
	if err != nil {
		panic(err)
	}
}

func (c CROSSConfig) importData() {
	importData := c.Actions.ImportData
	if importData == nil {
		return
	}
	if len(importData) == 0 {
		return
	}

	for i := 0; i < len(importData); i++ {
		commands := defaultDBToolCommand(c, true)
		id := importData[i]
		if !id.Run {
			continue
		}
		commands = append(commands, "importData")
		commands = append(commands, "--project-name="+ProjectName)
		commands = append(commands, "-n="+id.ProbeName)
		commands = append(commands, "-f="+id.ProbeXML)

		color.HiMagenta("Running command: " + strings.Join(commands, " "))
		err := c.executeCommand("java", commands, "")
		if err != nil {
			panic(err)
		}
	}
}

func (c CROSSConfig) runProbes() {
	probes := c.Actions.Probes

	if len(probes) == 0 {
		return
	}

	color.HiMagenta("Running probes command")

	for i := 0; i < len(probes); i++ {
		if !probes[i].Run {
			continue
		}
		commands := defaultDBToolCommand(c, true)
		probeName := probes[i].ProbeName
		commands = append(commands, "probe")
		commands = append(commands, "create")
		jarPath := c.getProbeJarPath(probeName, probes[i].JarPath)
		commands = append(commands, "-jar="+jarPath)
		commands = append(commands, "-n="+strings.ToUpper(probeName))
		commands = append(commands, "--project-name="+ProjectName)
		commands = append(commands, "--user-name="+Username)

		color.Magenta("\t create probe " + probeName + " command: " + strings.Join(commands, " "))
		err := c.executeCommand("java", commands, "")
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		runCommands := defaultDBToolCommand(c, true)
		runCommands = append(runCommands, "probe")
		runCommands = append(runCommands, "run")
		runCommands = append(runCommands, "-n="+strings.ToUpper(probeName))

		color.Magenta("\t run probe " + probeName + " command: " + strings.Join(runCommands, " "))
		err = c.executeCommand("java", runCommands, probeName)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

}

func (c CROSSConfig) getProbeJarPath(probeName, path string) string {
	repo := c.Paths.Repo
	return repo + path + "/target/" + probeName + ".jar"
}

func (c CROSSConfig) executeCommand(command string, args []string, filename string) error {
	cmd := exec.Command(command, args...)

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		return err
	}

	var f *os.File
	var errFile error

	if filename != "" {
		filepath := path.Join(c.Paths.LogsPath, filename+".txt")
		f, errFile = os.Create(filepath)
		if errFile != nil {
			fmt.Println(errFile)
		}
	}

	if c.PrintExecution {
		scanner := bufio.NewScanner(cmdReader)
		go func() {
			for scanner.Scan() {
				text := scanner.Text()
				if f != nil {
					f.WriteString(text + "\n")
				}
				color.Yellow("\t > %s\n", text)
			}
		}()
	}

	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error creating StderrPipe for Cmd", err)
		return err
	}

	errScanner := bufio.NewScanner(cmdErr)
	go func() {
		for errScanner.Scan() {
			color.Red("\t > %s\n", errScanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		return err
	}
	return nil
}

func defaultDBToolCommand(c CROSSConfig, appendProperties bool) []string {
	vec := strings.Split(c.Paths.DBToolPath, "/")
	dbToolName := vec[len(vec)-1]

	cmds := []string{
		"-jar",
		dbToolName + ".jar",
	}
	if appendProperties {
		cmds = append(cmds, "-p="+c.Paths.Repo+c.Paths.PropertiesFilePath)
	}
	return cmds
}
