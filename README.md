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

`leto-cli` is distributed via the snap `fort-leto-cli`

```bash
sudo snap install fort-leto-cli
sudo snap alias fort-leto-cli leto-cli
```

### TCP look up errors

you may certainly run into a `tcp lookup error` when performing any
command that accesses the network (`scan`,`start`,`stop` ...). This is
due to a limitation of snap regarding `.local` network mDNS addresses. It
can be solved using the following commands once.

``` bash
sudo apt install nscd
sudo service snapd restart
```

In some cases, you will also to restart your system to clear theses
errors.

### Bash completion utility

The `fort-leto-cli` will install all completion script for your
shell. Previous version of `leto-cli` were asking you to install
manually completion definition in your `.bashrc`. Those old completion
will conflict with snap's one, so you would need to manually edit your
bashrc to remove all reference to the function `_leto_cli_completion()`


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
