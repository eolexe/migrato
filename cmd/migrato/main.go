package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	_ "github.com/gemnasium/migrate/driver/bash"
	_ "github.com/gemnasium/migrate/driver/cassandra"
	_ "github.com/gemnasium/migrate/driver/mysql"
	_ "github.com/gemnasium/migrate/driver/postgres"
	_ "github.com/gemnasium/migrate/driver/sqlite3"
	"github.com/gemnasium/migrate/file"
	"github.com/gemnasium/migrate/migrate"
	"github.com/gemnasium/migrate/migrate/direction"
	pipep "github.com/gemnasium/migrate/pipe"
	"gopkg.in/yaml.v2"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

const (
	cmdCreate  = "create"
	cmdMigrate = "migrate"
	cmdHelp    = "help"
	cmdVersion = "version"
	cmdUp      = "up"
	cmdDown    = "down"
	cmdReset   = "reset"
	cmdRedo    = "redo"
	cmdGoto    = "goto"
)

type consul struct{
	endpoint string
	path string
	configType string
}

func (c consul) IsConfigured() bool {
	return (c.endpoint != "") && (c.path != "") && (c.configType != "")
}

var config struct {
	defaultPath string
	envPath     string
	section   string

	consul consul
}

func main() {
	flag.Usage = func() {
		helpCmd()
	}
	flag.StringVar(&config.defaultPath, "def", "env/default.yml", "the default configuration file.")
	flag.StringVar(&config.envPath, "env", "env/docker.yml", "the environment configuration file.")
	flag.StringVar(&config.section, "section", "db", "the section from config file")

	flag.StringVar(&config.consul.endpoint,"consul-endpoint", "", "Consul endpoint.")
	flag.StringVar(&config.consul.path, "consul-path", "", "Configuration key in Consul.")
	flag.StringVar(&config.consul.configType, "consul-config-type", "json", "Consul configuration type (json, yaml).")
	flag.Parse()

	if config.consul.endpoint == "" {
		config.consul.endpoint = os.Getenv("CONSUL_ENDPOINT")
	}

	if config.consul.path == "" {
		config.consul.path = os.Getenv("CONSUL_PATH")
	}


	fmt.Println("--------------------------------")
	fmt.Println("Def config -", config.defaultPath)
	fmt.Println("Env config -", config.envPath)
	fmt.Println("-----------------")
	fmt.Println("Consul Path - ", config.consul.path)
	fmt.Println("Consul Endpoint - ", config.consul.endpoint)
	fmt.Println("Consul Configuration Type - ", config.consul.configType)
	fmt.Println("-----------------")

	fmt.Println("Section -", config.section)
	fmt.Println("--------------------------------")
	fmt.Println()

	cmd := flag.Arg(0)

	var c map[string]interface{}
	var err error

	if config.consul.IsConfigured() {
		c, err = readConfigFomConsul(config.consul, config.section)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		c, err = readConfigFromPath(config.defaultPath, config.envPath, config.section)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Validation parameters
	switch {
	case c["migration_dir"] == nil:
		fmt.Println("Error: please set 'migration_dir' parameter in DB config")
		return
	case c["driver"] == nil:
		fmt.Println("Error: please set 'driver' parameter in DB config")
		return
	case c["user"] == nil:
		fmt.Println("Error: please set 'user' parameter in DB config")
		return
	case c["password"] == nil:
		c["password"] = ""
	case c["address"] == nil:
		fmt.Println("Error: please set 'address' parameter in DB config")
		return
	case c["name"] == nil:
		fmt.Println("Error: please set 'name' parameter in DB config")
		return
	}

	driver := c["driver"].(string)
	var url string

	switch driver {
	case "postgres":
		address := c["address"].(map[string]interface{})
		host := address["host"].(string)
		port := address["port"].(float64)
		portString := strconv.FormatFloat(port, 'f', 0, 64)
		if err != nil {
			log.Fatal("Couldn't parse port")
		}

		url = fmt.Sprintf("%s://%s:%s@%s/%s?sslmode=disable", driver,
			c["user"].(string),
			c["password"].(string),
			host + ":"+ portString,
			c["name"].(string))

	case "mysql":
		address := c["address"].(map[string]interface{})
		host := address["host"].(string)
		port := address["port"].(float64)
		portString := strconv.FormatFloat(port, 'f', 0, 64)
		if err != nil {
			log.Fatal("Couldn't parse port")
		}

		url = fmt.Sprintf("%s://%s:%s@tcp(%s)/%s", driver,
			c["user"].(string),
			c["password"].(string),
			host + ":"+ portString,
			c["name"].(string))
	default:
		fmt.Printf("Error: unknown driver '%s'\n", driver)
		return

	}

	migrationDir := c["migration_dir"].(string)
	switch cmd {
	case cmdCreate:

		name := flag.Arg(1)
		if name == "" {
			fmt.Println("Please specify name.")
			os.Exit(1)
		}

		migrationFile, err := migrate.Create(url, migrationDir, name)
		switch e := err.(type) {
		case *net.OpError:
			fmt.Println(fmt.Sprintf("Can't connect to the DB: %s. Error: %s", url, e))
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Version %v migration files created in %v:\n", migrationFile.Version, migrationDir)
		fmt.Println(migrationFile.UpFile.FileName)
		fmt.Println(migrationFile.DownFile.FileName)
		////////// CREATE END ///////////////
	case cmdMigrate:
		relativeN := flag.Arg(1)
		relativeNInt, err := strconv.Atoi(relativeN)
		if err != nil {
			fmt.Println("Unable to parse param <n>.")
			os.Exit(1)
		}
		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Migrate(pipe, url, migrationDir, relativeNInt)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}
		////////// MIGRATE END ///////////////
	case cmdUp:
		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Up(pipe, url, migrationDir)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}
		////////// UP END ///////////////
	case cmdDown:
		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Down(pipe, url, migrationDir)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}

		////////// DOWN END ///////////////
	case cmdVersion:
		version, err := migrate.Version(url, migrationDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(version)
		////////// VERSION END ///////////////

	case cmdReset:
		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Reset(pipe, url, migrationDir)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}
		////////// RESET END ///////////////

	case cmdRedo:
		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Redo(pipe, url, migrationDir)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}
		////////// REDO END ///////////////

	case cmdGoto:
		toVersion := flag.Arg(1)
		toVersionInt, err := strconv.Atoi(toVersion)
		if err != nil || toVersionInt < 0 {
			fmt.Println("Unable to parse param <v>.")
			os.Exit(1)
		}

		currentVersion, err := migrate.Version(url, migrationDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		relativeNInt := toVersionInt - int(currentVersion)

		timerStart = time.Now()
		pipe := pipep.New()
		go migrate.Migrate(pipe, url, migrationDir, relativeNInt)
		ok := writePipe(pipe)
		printTimer()
		if !ok {
			os.Exit(1)
		}
		////////// GOTO  END ///////////////

	default:
		fallthrough
	case cmdHelp:
		helpCmd()
	}
}

func writePipe(pipe chan interface{}) (ok bool) {
	okFlag := true
	if pipe != nil {
		for {
			select {
			case item, more := <-pipe:
				if !more {
					return okFlag
				}

				switch item.(type) {
				case string:
					fmt.Println(item.(string))

				case error:
					c := color.New(color.FgRed)
					c.Println(item.(error).Error())
					okFlag = false

				case file.File:
					f := item.(file.File)
					c := color.New(color.FgBlue)
					if f.Direction == direction.Up {
						c.Print(">")
					} else if f.Direction == direction.Down {
						c.Print("<")
					}
					fmt.Printf(" %s\n", f.FileName)

				default:
					text := fmt.Sprint(item)
					fmt.Println(text)
				}
			}
		}
	}
	return okFlag
}

var timerStart time.Time

func printTimer() {
	diff := time.Now().Sub(timerStart).Seconds()
	if diff > 60 {
		fmt.Printf("\n%.4f minutes\n", diff/60)
	} else {
		fmt.Printf("\n%.4f seconds\n", diff)
	}
}

func helpCmd() {
	os.Stderr.WriteString(
		`usage: migrato [-def=<path> -env=<path>] <command> [<args>]
Commands:
   create <name>  Create a new migration
   up             Apply all -up- migrations
   down           Apply all -down- migrations
   reset          Down followed by Up
   redo           Roll back most recent migration, then apply it again
   version        Show current migration version
   migrate <n>    Apply migrations -n|+n
   help           Show this help
'-path' defaults to the subdirectory env of current working directory.
`)
}


func readConfigFomConsul(c consul, section string) (map[string]interface{}, error) {
	v := viper.New()
	v.AddRemoteProvider("consul", c.endpoint, c.path)
	v.WatchRemoteConfig()
	v.SetConfigType(c.configType)

	var err error
	tries := 10
	for tries > 0 {
		if err = v.ReadRemoteConfig(); err != nil {
			time.Sleep(200 * time.Millisecond)
			tries--
			log.Fatal("Couldn't read config, retrying... (%s)\n", err)
		} else {
			break
		}
	}

	if tries == 0 {
		log.Fatalf("Could not read config file: %s\n", err)
	}

	config := map[string]interface{}{}

	err = v.Unmarshal(&config)

	sc, ok := config[section]
	if !ok {
		log.Fatalf("Could not get %s section from consul config \n", section)
	}

	sectionConfig, ok := sc.(map[string]interface{})
	if !ok {
		log.Fatalf("Could not get %s section from consul config \n", section)
	}

	return sectionConfig, err
}

func readConfigFromPath(defaultConfigPath, envConfigPath, section string) (map[string]interface{}, error) {
	def, err := ioutil.ReadFile(defaultConfigPath)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(def)

	env, err := ioutil.ReadFile(envConfigPath)
	if err != nil {
		return nil, err
	}

	buf.Write(env)

	cMap := map[string]map[string]interface{}{}

	if err = yaml.Unmarshal(buf.Bytes(), cMap); err != nil {
		return nil, err
	}

	if _, ok := cMap["config"][section]; !ok {
		return nil, fmt.Errorf("Section in config is undefined")
	}

	getSectionBytes, err := yaml.Marshal(cMap["config"][section])
	if err != nil {
		return nil, err
	}

	sectionMap := map[string]interface{}{}

	if err = yaml.Unmarshal(getSectionBytes, sectionMap); err != nil {
		return nil, err
	}

	return sectionMap, nil
}
