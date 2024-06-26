Project Norma
=============

Project of integrating Carmen Storage and Tosca VM into go-opera.

# Building and Running

## Requirements

For building/running the project, the following tools are required:
* Go: version 1.20 or later; we recommend to use your system's package manager; alternatively, you can follow Go's [installation manual](https://go.dev/doc/install) or; if you need to maintain multiple versions, [this tutorial](https://go.dev/doc/manage-install) describes how to do so
* Docker: version 23.0 or later; we recommend to use your system's package manager or the installation manuals listed in the [Using Docker](#using-docker) section below
* GNU make, or compatible
* [R](https://www.r-project.org/): make sure the command `Rscript` is available on your system.
  * To install R and all needed dependencies on Ubuntu, use `sudo apt install r-base-core pandoc libcurl4-openssl-dev libssl-dev libfontconfig1-dev libharfbuzz-dev libfribidi-dev libfreetype6-dev libpng-dev libtiff5-dev libjpeg-dev`
  * To install R packages manually (may be necessary for first-time R usage), start an R session by running the command `R`, and run the command `install.packages(c("rmarkdown", "tidyverse", "lubridate", "slider"))` inside the R session. You may be prompted to create a user-specific directory for library dependencies. If so, confirm this.

Optionally, before running `make generate-mocks`, make sure you installed:
* GoMock: `go install github.com/golang/mock/mockgen@v1.6.0`
  * Make sure `$GOPATH/bin` is in your `$PATH`. `$GOPATH` defaults to `$HOME/go` if not set, i.e. configure `$PATH` 
  * either to `PATH=$GOPATH/bin:$PATH` or `PATH=$HOME/go/bin:$PATH` 

Optionally, before running `make generate-abi`, make sure you have installed:
* Solidity Compiler (solc) - see [Installing the Solidity Compiler](https://docs.soliditylang.org/en/latest/installing-solidity.html)
  * Install version [0.8.19](https://github.com/ethereum/solidity/releases/tag/v0.8.19)
* go-ethereum's abigen: 
  * Checkout [go-ethereum](https://github.com/ethereum/go-ethereum/) `git clone https://github.com/ethereum/go-ethereum/`
  * Checkout the right version `git checkout v1.10.8`
  * Build Geth will all tools: `cd go-ethereum` and `make all`
  * Copy `abigen` from `build/bin/abigen` into your PATH, e.g.: `cp build/bin/abigen /usr/local/bin`


## Building

To build the project, run
```
make -j
```
This will build the required docker images (make sure you have Docker access permissions!) and the Norma go application. To run tests, use
```
make test
```
To clean up a build, use `make clean`.

## Running

To run Norma, you can run the `norma` executable created by the build process:
```
build/norma <cmd> <args...>
```
To list the available commands, run
```
build/norma
```


# Developer Information

## Using Docker

Some experiments simulate network using Docker. For a local development the Docker must be installed:
* MacOS: https://docs.docker.com/desktop/install/mac-install/
* Linux: https://docs.docker.com/engine/install/ubuntu/

### Permissions on Linux
After installation, make sure your user has the needed permissions to run docker containers on your system. You can test this by running
```
docker images
```
If you get an error stating a lack of permissions, you might have to add your non-root user to the docker group (see [this stackoverflow post](https://stackoverflow.com/questions/48957195/how-to-fix-docker-got-permission-denied-issue) for details):
```
sudo groupadd docker
sudo usermod -aG docker $USER
newgrp docker
```
If the `newgrp docker` command is not working, a `reboot` might help.

### Docker Sock on MacOS
If Norma tests produce error that Docker is not listening on  `unix:///var/run/docker.sock`, execute
* `docker context inspect` and make note of `Host`, which should be `unix:///$HOME/.docker/run/docker.sock`
* export system variable, i.e. add to either `/etc/zprofile` or `$HOME/.zprofile`: 
* `export DOCKER_HOST=unix:///$HOME/.docker/run/docker.sock`

alternatively
* Open `Desktop Tool` --> `Settings` --> `Advanced` --> `Enable Default Docker socket`
  * this will bind the docker socket to default `unix:///var/run/docker.sock`


### Building
The experiments use the docker image that wraps the forked Opera/Norma client. The image is build as part of 
the build process, and can be explicitly triggered:
```
make build-docker-image
```

### Commands
During the development, a few Docker commands can come handy:
```
docker run -i -t -d opera         // runs container with Opera in background (without -d it would run in foreground)
docker ps                         // shows running container
docker exec -it <ID> /bin/sh      // opens interactive shell inside the container, the ID is obtained by previous command
docker logs <ID>                  // prints stdout (log) of the container
docker stop <ID>                  // stop (kills) the container
docker rm -f $(docker ps -a -q)   // stop and clean everything 
```

# Analyzing Build-In Metrics

Norma manages and observes a network of Opera nodes and collects a set of metrics. The metrics are automatically enabled and their outcome is stored in a CSV file, which allows for later processing in spreadsheet software. 

For instance, metric data can be generated by just running the example scenario:

```
build/norma run scenarios/small.yml 
```

which produces a directory filled with measurment results, which is printed at the end of the application output. Look for two lines like

```
Monitoring data was written to /tmp/norma_data_<random_number>
Raw data was exported to /tmp/norma_data_<random_number>/measurements.csv
```

The first line lists the directory in which all monitoring data was written to. This, in particular, includes the `measurements.csv` file, containing most of the collected monitoring data in a CSV format. It merges all the metrics in one file, and every line is one result of a single meassurement. The header of the file is:
```
| Metric | Network | Node | App | Time | Block | Workers | Value |
```
* Metric -- is the string name of the metric
* Network -- is the network name, currently always the same
* Node -- if the metric is attached to a node, the name is shows, otherwise the column is empty
* App -- if the metric is attached to an application (smart contract), the name is shows, otherwise the column is empty
* Time -- if the metric is meassured for time series (i.e. time on X-axis), the timestamp is provided, otherwise the column is empty
* Block -- if the metric is meassured for block series (i.e. block height on X-axis), the block number is provided, otherwise the column is empty
* Workers -- if the metric is meassured for the number of workers sending transactions (i.e. the number of workers on X-axis), the number is provided, otherwise the column is empty
* Value -- this column is always filled and contains the actual valu (i.e. Y-axis) meassured for the metrcis. 

It means that Metrics can meassure values for block numbers or timeseries, and it can be done for the whole network, individual nodes, or applications. The metrics are all stored in the same file and values 
that do not apply for particular metric are left empty. 

This structure allows for easily filtering metrics of interest and importing them in a unified format to a spreadshead. The rows oriented format can be turned into rows/cells format using a Pivot table. 

For instance, lets analyse the transaction throughput of the nodes. List the metric using grep:
```
grep TransactionsThroughput output.csv 
```
or directly store the result to the clipboard (MacOS)
```
grep TransactionsThroughput output.csv | pbcopy
```

The content of the clipboard can be inserted into Google Sheet. For the Pivot table to work, the header must be in the first row. 
When the rows are inserted, it must be clicked to `Split text column`, then the data is ready:

<img width="875" alt="image" src="https://github.com/Fantom-foundation/Norma/assets/7114574/29b51cf6-8d9e-44d6-a1e1-d8984902451e">

Notice that the CSV file could be inserted as whole (`cat output.csv | pbcopy`) to have all metrics at hand for the analysis.
This can be impossible for same large files though, as for instance a spreadsheet tool can become unresponsive.    

To create the Pivot table, one has to click: `Insert -> Pivot table`, Select `data range` and Insert to a `New sheet`


<img width="853" alt="image" src="https://github.com/Fantom-foundation/Norma/assets/7114574/c3b8086b-37bb-49c8-bfdc-edff68a35ddb">
<img width="851" alt="image" src="https://github.com/Fantom-foundation/Norma/assets/7114574/9cc049b0-f919-44b2-bdd6-f5f5599b7cf7">

The new empty Pivot table will pop-up. What to show in the table depends on particular needs, but since the metric we have chosen contains the throughput 
of each node, meassured for block height, it is a good idea to have the nodes as columns and values as rows, the first row being the block number. It must be set:
* Rows: `Block` 
* Column: `Node`
* Values: `Value`

Notice that the items selected from drop down menus are actually the columns from the flat CSV file that has been imported. 

The metrics used actually contains three additional metricts, in total:
* `TransactionsThroughput` - transaction throughput for every block and node
* `TransactionsThroughputSMA_10` - simple moving average for 10 blocks
* `TransactionsThroughputSMA_100` - simple moving average for 100 blocks
* `TransactionsThroughputSMA_1000` - simple moving average for 1000 blocks

To see only metric of interest, one has to filter it in the `Filters` drop down menu. 

Notice that the Pivot table groups potentially clashing rows (like SQL GROUP BY) and applies a selected function such as Sum, Avr, Max, Min etc. At the moment we do not have metrics where such a grouping would make sense, i.e. it is imporrtant to enable filter just for one metric at a time, and then the applied grouping can be ignored (it cannot be dissabled).
Also it is good to uncheck `Show totals` for many metrics where the sums do not make sense. 

As a last step, charts can be plot from the data as usual, like this:


<img width="2413" alt="image" src="https://github.com/Fantom-foundation/Norma/assets/7114574/5aeba3ec-a7ea-4b67-9aa4-12453bd149e6">


## CPU Profile Data

In addition to the Norma metrics, the `pprof` CPU proifile is collected every 10s from each node. The profiles are stored in the temp directory. The directory name is printed together with the Norma output, for instance:

```
Monitoring data was written to /tmp/norma_data_1852477583
```

The directory has the following structure:
```
/tmp/norma_data_<rand>
+ - cpu_profiles
  + - <node-name>
    + - <sample_number>.prof
    | - <sample_number>.prof
      ...
```
These files can be transfered to a developer's machine, and analysed by running

```
go tool pprof -http=":8000" <sample_number>.prof
```

## Known Norma Restrictions

Known restrictions
 - only one node will be a validator, and it is the first node to be started; this node must life until the end of the scenario
 - currently, all transactions are send to the validator node
