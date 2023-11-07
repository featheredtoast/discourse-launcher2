package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/discourse/discourse_docker/discourse-builder/config"
	"github.com/discourse/discourse_docker/discourse-builder/docker"
	"github.com/discourse/discourse_docker/discourse-builder/utils"
	"os"
	"os/exec"
)

type StartCmd struct {
	Config     string `arg:"" name:"config" help:"config"`
	DryRun     bool   `name:"dry-run" short:"n" help:"print start command only"`
	DockerArgs string `name:"docker-args" help:"Extra arguments to pass when running docker"`
	RunImage   string `name:"run-image" help:"Override the image used for running the container"`
	Supervised bool   `name:"supervised" help:"Supervised run"`
}

// TODO: commands started here cannot be stopped unless forced???
func (r *StartCmd) Run(cli *Cli, ctx *context.Context) error {
	//start stopped container first if exists
	running, _ := docker.ContainerRunning(r.Config)
	if running {
		fmt.Println("Nothing to do, your container has already started!")
		return nil
	}
	exists, _ := docker.ContainerExists(r.Config)
	if exists {
		fmt.Println("starting up existing container")
		cmd := exec.CommandContext(*ctx, "docker", "start", r.Config)
		fmt.Println(cmd)
		if err := utils.CmdRunner(cmd).Run(); err != nil {
			return err
		}
		return nil
	}

	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return errors.New("YAML syntax error. Please check your containers/*.yml config files.")
	}
	defaultHostname, _ := os.Hostname()
	defaultHostname = defaultHostname + "-" + r.Config
	hostname := config.DockerHostname(defaultHostname)

	restart := true
	detatch := true
	if r.Supervised {
		restart = false
		detatch = false
	}
	//TODO: implement
	runner := docker.DockerRunner{
		Config:      config,
		Ctx:         ctx,
		ContainerId: r.Config,
		DryRun:      r.DryRun,
		CustomImage: r.RunImage,
		Restart:     restart,
		Detatch:     detatch,
		ExtraFlags:  r.DockerArgs,
		Hostname:    hostname,
	}
	return runner.Run()
}

type RunCmd struct {
	RunImage   string   `name:"run-image" help:"Override the image used for running the container"`
	DockerArgs string   `name:"docker-args" help:"Extra arguments to pass when running docker"`
	Config     string   `arg:"" name:"config" help:"config"`
	Cmd        []string `arg:"" help:"command to run" passthrough:""`
}

func (r *RunCmd) Run(cli *Cli, ctx *context.Context) error {
	//TODO: implement
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return errors.New("YAML syntax error. Please check your containers/*.yml config files.")
	}
	runner := docker.DockerRunner{
		Config:      config,
		Ctx:         ctx,
		CustomImage: r.RunImage,
		SkipPorts:   true,
		Rm:          true,
		ExtraFlags:  r.DockerArgs,
		Cmd:         r.Cmd,
	}
	return runner.Run()
	return nil
}

type StopCmd struct {
	Config string `arg:"" name:"config" help:"config"`
}

func (r *StopCmd) Run(cli *Cli, ctx *context.Context) error {
	exists, _ := docker.ContainerExists(r.Config)
	if !exists {
		fmt.Println(r.Config + "was not found")
		return nil
	}
	cmd := exec.CommandContext(*ctx, "docker", "stop", "-t", "600", r.Config)
	fmt.Println(cmd)
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	return nil
}

type RestartCmd struct {
	Config     string `arg:"" name:"config" help:"config"`
	DockerArgs string `name:"docker-args" help:"Extra arguments to pass when running docker"`
	RunImage   string `name:"run-image" help:"Override the image used for running the container"`
}

func (r *RestartCmd) Run(cli *Cli, ctx *context.Context) error {
	start := StartCmd{Config: r.Config, DockerArgs: r.DockerArgs, RunImage: r.RunImage}
	stop := StopCmd{Config: r.Config}
	if err := stop.Run(cli, ctx); err != nil {
		return err
	}
	if err := start.Run(cli, ctx); err != nil {
		return err
	}
	return nil
}

type DestroyCmd struct {
	Config string `arg:"" name:"config" help:"config"`
}

func (r *DestroyCmd) Run(cli *Cli, ctx *context.Context) error {
	exists, _ := docker.ContainerExists(r.Config)
	if !exists {
		fmt.Println(r.Config + "was not found")
		return nil
	}

	cmd := exec.CommandContext(*ctx, "docker", "stop", "-t", "600", r.Config)
	fmt.Println(cmd)
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	cmd = exec.CommandContext(*ctx, "docker", "rm", r.Config)
	fmt.Println(cmd)
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	return nil
}

type EnterCmd struct {
	Config string `arg:"" name:"config" help:"config"`
}

func (r *EnterCmd) Run(cli *Cli, ctx *context.Context) error {
	cmd := exec.CommandContext(*ctx, "docker", "exec", "-it", r.Config, "/bin/bash", "--login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	return nil
}

type LogsCmd struct {
	Config string `arg:"" name:"config" help:"config"`
}

func (r *LogsCmd) Run(cli *Cli, ctx *context.Context) error {
	cmd := exec.CommandContext(*ctx, "docker", "logs", r.Config)
	output, err := utils.CmdRunner(cmd).Output()
	if err != nil {
		return err
	}
	fmt.Println(string(output[:]))
	return nil
}

type RebuildCmd struct {
	Config     string `arg:"" name:"config" help:"config"`
	DockerArgs string `name:"docker-args" help:"Extra arguments to pass when running docker"`
}

func (r *RebuildCmd) Run(cli *Cli, ctx *context.Context) error {
	//TODO implement
	return nil
}

type CleanupCmd struct{}

func (r *CleanupCmd) Run(cli *Cli, ctx *context.Context) error {
	cmd := exec.CommandContext(*ctx, "docker", "container", "prune", "--filter", "until=1h")
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	cmd = exec.CommandContext(*ctx, "docker", "image", "prune", "--all", "--filter", "until=1h")
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	_, err := os.Stat("/var/discourse/shared/standalone/postgres_data_old")
	if !os.IsNotExist(err) {
		fmt.Println("Old PostgreSQL backup data cluster detected")
		fmt.Println("Would you like to remove it? (y/N)")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		reply := scanner.Text()
		if reply == "y" || reply == "Y" {
			fmt.Println("removing old PostgreSQL data cluster at /var/discourse/shared/standalone/postgres_data_old...")
			os.RemoveAll("/var/discourse/shared/standalone/postgres_data_old")
		} else {
			return errors.New("Cancelled")
		}
	}

	return nil
}