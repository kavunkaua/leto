# Leto: Vision Tacking management for the FORmicidae Tracker

This directory contains two tools :
 * `leto` : a tool used to manage ann artemis process for the tracking
   of ants, it is installed on atracking computer as a service. Its
   administration is managed by the [FORT ansible
   script](https://github.com/formicidae-tracker/fort-configuration).
 *  `leto-cli` a small command line tool that could be installed on
    any computer and used to manage leto instances on a local network.

## Leto-cli installation and upgrade

Simply use the `go get` command:

```bash
go get -u github.com/formicidae-tracker/leto/leto-cli
```

The `-u` is for asking to fetch and install the latest sources.
