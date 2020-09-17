# Leto: Vision Tacking management for the FORmicidae Tracker

This directory contains two tools :
 *  `leto-cli` a small command line tool that could be installed on
    any computer and used to manage leto instances on a local
    network. This is the tool most user want to install.
 * `leto` : a tool used to manage ann artemis process for the tracking
   of ants, it is installed on atracking computer as a service. Its
   administration is managed by the [FORT ansible
   script](https://github.com/formicidae-tracker/fort-configuration). Apart
   for system administrator user should not install nor use it
   directly.

## Leto-cli installation and upgrade

Simply use the `go get` command:

```bash
go get -u github.com/formicidae-tracker/leto/leto-cli
```

The `-u` is for asking to fetch and install the latest sources.

### Bash completion utility

The `leto-cli` comes with a useful bash completion script. With it wou
will be able to auto-complete command names and arguments, but also
the online node name.

In order to enable the bash completion please append the content of
the file
[leto-cli/misc/bash/leto-cli-complete.bash](leto-cli/misc/bash/leto-cli-complete.bash)
to your `.bashrc`. You can do it by running once the following
snippet:

``` bash
wget -O /tmp/leto-cli-complete.bash https://raw.githubusercontent.com/formicidae-tracker/leto/master/leto-cli/misc/bash/leto-cli-complete.bash
cat /tmp/leto-cli-complete.bash >> ~/.bashrc
```

## `leto-cli` Usage

There are a few `leto-cli` commands

 * `leto-cli scan` scans all availables nodes on the local network and
   displays their status
 * `leto-cli start nodename [OPTIONS] [configFile]`: starts an
   experiment on node `nodename` with either command line options or
   using a yaml `configFile`.
 * `leto-cli stop nodename`: stops any experiment on `nodename`
 * `leto-cli status nodename`: displays current status for `nodename`,
   like current experiment configuration and output directory
 * `leto-cli last-experiment-log nodename`: displays the log of the
   last **finished** experiment on `nodename`, with its original
   configuration and artemis complete logs
 * `leto-cli display-frame-readout nodename`: displays a live stream
   data of currnet number of detected tags and quads on the running
   node
