# Server Backup Simple

App to be used to backup remote server including file folders using rsync and backup of postgresql database using ssh and pg_dump on remote machine.

# How to use

## Compile the project using the command bellow.
```
# For linux
./build

```

## Create config file
Create the folder dist/config and copy the file dist/config_example.json for it app you want to backup changing the data on it

## Install service

```
./install
```