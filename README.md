# db-apex-export

Command to connect to an Oracle Database and export an Application Express application

## Prerequisites

Oracle Instant Client must already be installed

[Oracle Instant Client](https://www.oracle.com/database/technologies/instant-client.html)

Note - Oracle Instant Client must be configured per your environment (please follow the instructions provided by Oracle).

## Table of Contents

- [db-apex-export](#db-apex-export)
  - [Prerequisites](#Prerequisites)
  - [Table of Contents](#Table-of-Contents)
  - [Installation](#Installation)
  - [Building](#Building)
  - [Usage](#Usage)
  - [Support](#Support)
  - [Contributing](#Contributing)

## Installation

1) Clone this repository into a local directory, copy the db-apex-export executable into your $PATH

```bash
$ git clone https://github.com/apexevangelists/db-apex-export
```

## Building

Pre-requisite - install Go

Compile the program -

```bash
$ go build
```

## Usage

```bash-3.2$ ./db-apex-export -h
Usage of ./db-apex-export:
  -appId string
    	Application ID to export (specify multiple separated by a comma)
  -configFile string
    	Configuration file for general parameters (default "config")
  -connection string
    	Configuration file for connection
  -db string
    	Database Connection, e.g. user/password@host:port/sid
  -debug
    	Debug mode (default=false)
  -o string
    	Filename used for the export file (specify multiple separated by a comma)

bash-3.2$
```

## Support

Please [open an issue](https://github.com/apexevangelists/db-apex-export/issues/new) for support.

## Contributing

Please contribute using [Github Flow](https://guides.github.com/introduction/flow/). Create a branch, add commits, and [open a pull request](https://github.com/apexevangelists/db-apex-export/compare).