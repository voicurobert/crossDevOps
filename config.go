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
}

type Probe struct {
	ProbeName string `mapstructure:"probeName"`
	JarPath   string `mapstructure:"jarPath"`
}

type Actions struct {
	CreateDB      bool     `mapstructure:"createDB"`
	Initialize    bool     `mapstructure:"initialize"`
	ImportConfigs []string `mapstructure:"importConfigs"`
	CreateProject bool     `mapstructure:"createProject"`
	ProbesToRun   []string `mapstructure:"probesToRun"`
}

type CROSSConfig struct {
	PrintExecution bool    `mapstructure:"printExecution"`
	Paths          Paths   `mapstructure:"paths"`
	Probes         []Probe `mapstructure:"probes"`
	Actions        Actions `mapstructure:"actions"`
}

func (c CROSSConfig) RunActions() {
	c.goToDBToolPath()
	c.createDB()
	c.initialize()
	c.importConfigCommands()
	c.createProject()
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
		commands = append(commands, "importConfig")
		if strings.Split(cmd, "")[0] != "-" {
			commands = append(commands, "-f")
		}
		commands = append(commands, cmd)

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

func (c CROSSConfig) runProbes() {
	probes := c.Actions.ProbesToRun

	if len(probes) == 0 {
		return
	}

	color.HiMagenta("Running probes command")

	for i := 0; i < len(probes); i++ {
		commands := defaultDBToolCommand(c, true)
		probeName := probes[i]
		commands = append(commands, "probe")
		commands = append(commands, "create")
		jarPath := c.getProbeJarPath(probeName)
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

func (c CROSSConfig) getProbeJarPath(probeName string) string {
	repo := c.Paths.Repo
	for _, probe := range c.Probes {
		if probe.ProbeName != probeName {
			continue
		}
		return repo + probe.JarPath + "/target/" + probeName + ".jar"
	}
	return ""
}

func (c CROSSConfig) executeCommand(command string, args []string, filename string) error {
	cmd := exec.Command(command, args...)

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		return err
	}

	if c.PrintExecution {
		scanner := bufio.NewScanner(cmdReader)
		go func() {
			for scanner.Scan() {
				if filename != "" {
					dir, _ := os.Getwd()
					filepath := path.Join(dir, filename, ".txt")
					f, _ := os.Create(filepath)
					f.WriteString(scanner.Text())
				}
				color.Yellow("\t > %s\n", scanner.Text())
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
