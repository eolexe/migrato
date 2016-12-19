# migrato - Migration Hero
SQL Migration Tool that uses our config files.

### Installation

``` go get github.com/eolexe/migrato ```

### Usage

You need to be in the project root directory (migrato search for the config files in env/ folder) to execute the command.

you can change this behavior by specified the path to the default and env config file, using -def and -env flags.


```shell
$ migrato create InitDatabase
Version 2 migration files created in db/migrations:
20161213134301_InitDatabase.up.sql
20161213134301_InitDatabase.down.sql

$ tree db/migrations
db/migrations
├── 20161213134301_InitDatabase.down.sql
├── 20161213134301_InitDatabase.up.sql

$ migrato up
> 20161213134301_InitDatabase.up.sql

0.0677 seconds

$ migrato down
< 20161213134301_InitDatabase.down.sql

0.0751 seconds

```


This project is based on [Migrate](https://github.com/gemnasium/migrate) Lib.
