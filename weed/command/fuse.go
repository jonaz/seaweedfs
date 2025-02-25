package command

import (
	"fmt"
	"strings"
	"strconv"
	"time"
	"os"
)

func init() {
	cmdFuse.Run = runFuse // break init cycle
}

type parameter struct {
	name string
	value string
}

func runFuse(cmd *Command, args []string) bool {
	rawArgs := strings.Join(args, " ")
	rawArgsLen := len(rawArgs)
	option := strings.Builder{}
	options := []parameter{}
	masterProcess := true

	// first parameter
	i := 0
	for i = 0; i < rawArgsLen && rawArgs[i] != ' '; i++ {
		option.WriteByte(rawArgs[i])
	}
	options = append(options, parameter{"arg0", option.String()})
	option.Reset()

	for i++; i < rawArgsLen; i++ {

		// space separator check for filled option
		if rawArgs[i] == ' ' {
			if option.Len() > 0 {
				options = append(options, parameter{option.String(), "true"})
				option.Reset()
			}

		// dash separator read option until next space
		} else if rawArgs[i] == '-' {
			for i++; i < rawArgsLen && rawArgs[i] != ' '; i++ {
				option.WriteByte(rawArgs[i])
			}
			options = append(options, parameter{option.String(), "true"})
			option.Reset()

		// equal separator start option with pending value
		} else if rawArgs[i] == '=' {
			name := option.String()
			option.Reset()

			for i++; i < rawArgsLen && rawArgs[i] != ','; i++ {
				// double quote separator read option until next double quote
				if rawArgs[i] == '"' {
					for i++; i < rawArgsLen && rawArgs[i] != '"'; i++ {
						option.WriteByte(rawArgs[i])
					}

				// single quote separator read option until next single quote
				} else if rawArgs[i] == '\'' {
					for i++; i < rawArgsLen && rawArgs[i] != '\''; i++ {
						option.WriteByte(rawArgs[i])
					}

				// add chars before comma
				} else if rawArgs[i] != ' ' {
					option.WriteByte(rawArgs[i])
				}
			}

			options = append(options, parameter{name, option.String()})
			option.Reset()

		// comma separator just read current option
		} else if rawArgs[i] == ',' {
			options = append(options, parameter{option.String(), "true"})
			option.Reset()

		// what is not a separator fill option buffer
		} else {
			option.WriteByte(rawArgs[i])
		}
	}

	// get residual option data
	if option.Len() > 0 {
		// add value to pending option
		options = append(options, parameter{option.String(), "true"})
		option.Reset()
	}

	// scan each parameter
	for i := 0; i < len(options); i++ {
		parameter := options[i]

		switch parameter.name  {
		case "child":
			masterProcess = false
		case "arg0":
			mountOptions.dir = &parameter.value
		case "filer":
			mountOptions.filer = &parameter.value
		case "filer.path":
			mountOptions.filerMountRootPath = &parameter.value
		case "dirAutoCreate":
			if parsed, err := strconv.ParseBool(parameter.value); err != nil {
				mountOptions.dirAutoCreate = &parsed
			} else {
				panic(fmt.Errorf("dirAutoCreate: %s", err))
			}
		case "collection":
			mountOptions.collection = &parameter.value
		case "replication":
			mountOptions.replication = &parameter.value
		case "disk":
			mountOptions.diskType = &parameter.value
		case "ttl":
			if parsed, err := strconv.ParseInt(parameter.value, 0, 32); err != nil {
				intValue := int(parsed)
				mountOptions.ttlSec = &intValue
			} else {
				panic(fmt.Errorf("ttl: %s", err))
			}
		case "chunkSizeLimitMB":
			if parsed, err := strconv.ParseInt(parameter.value, 0, 32); err != nil {
				intValue := int(parsed)
				mountOptions.chunkSizeLimitMB = &intValue
			} else {
				panic(fmt.Errorf("chunkSizeLimitMB: %s", err))
			}
		case "concurrentWriters":
			i++
			if parsed, err := strconv.ParseInt(parameter.value, 0, 32); err != nil {
				intValue := int(parsed)
				mountOptions.concurrentWriters = &intValue
			} else {
				panic(fmt.Errorf("concurrentWriters: %s", err))
			}
		case "cacheDir":
			mountOptions.cacheDir = &parameter.value
		case "cacheCapacityMB":
			if parsed, err := strconv.ParseInt(parameter.value, 0, 64); err != nil {
				mountOptions.cacheSizeMB = &parsed
			} else {
				panic(fmt.Errorf("cacheCapacityMB: %s", err))
			}
		case "dataCenter":
			mountOptions.dataCenter = &parameter.value
		case "allowOthers":
			if parsed, err := strconv.ParseBool(parameter.value); err != nil {
				mountOptions.allowOthers = &parsed
			} else {
				panic(fmt.Errorf("allowOthers: %s", err))
			}
		case "umask":
			mountOptions.umaskString = &parameter.value
		case "nonempty":
			if parsed, err := strconv.ParseBool(parameter.value); err != nil {
				mountOptions.nonempty = &parsed
			} else {
				panic(fmt.Errorf("nonempty: %s", err))
			}
		case "volumeServerAccess":
			mountOptions.volumeServerAccess = &parameter.value
		case "map.uid":
			mountOptions.uidMap = &parameter.value
		case "map.gid":
			mountOptions.gidMap = &parameter.value
		case "readOnly":
			if parsed, err := strconv.ParseBool(parameter.value); err != nil {
				mountOptions.readOnly = &parsed
			} else {
				panic(fmt.Errorf("readOnly: %s", err))
			}
		case "cpuprofile":
			mountCpuProfile = &parameter.value
		case "memprofile":
			mountMemProfile = &parameter.value
		case "readRetryTime":
			if parsed, err := time.ParseDuration(parameter.value); err != nil {
				mountReadRetryTime = &parsed
			} else {
				panic(fmt.Errorf("readRetryTime: %s", err))
			}
		}
	}

	// the master start the child, release it then finish himself
	if masterProcess {
		arg0 := os.Args[0]
		argv := append(os.Args, "-o", "child")

		attr := os.ProcAttr{}
		child, err := os.StartProcess(arg0, argv, &attr)

		if err != nil {
			panic(fmt.Errorf("master process can not start child process: %s", err))
		}

		err = child.Release()

		if err != nil {
			panic(fmt.Errorf("master process can not release child process: %s", err))
		}

		return true
	}

	// I don't know why PATH environment variable is lost
	if err := os.Setenv("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin"); err != nil {
		panic(fmt.Errorf("setenv: %s", err))
	}

	// just call "weed mount" command
	return runMount(cmdMount, []string{})
}

var cmdFuse = &Command{
	UsageLine: "fuse /mnt/mount/point -o \"filer=localhost:8888,filer.path=/\"",
	Short: "Allow use weed with linux's mount command",
	Long: `Allow use weed with linux's mount command

  You can use -t weed on mount command:
  mv weed /sbin/mount.weed
  mount -t weed fuse /mnt -o "filer=localhost:8888,filer.path=/"

  Or you can use -t fuse on mount command:
  mv weed /sbin/weed
  mount -t fuse.weed fuse /mnt -o "filer=localhost:8888,filer.path=/"
  mount -t fuse "weed#fuse" /mnt -o "filer=localhost:8888,filer.path=/"

  To use without mess with your /sbin:
  mount -t fuse./home/user/bin/weed fuse /mnt -o "filer=localhost:8888,filer.path=/"
  mount -t fuse "/home/user/bin/weed#fuse" /mnt -o "filer=localhost:8888,filer.path=/"

  To pass more than one parameter use quotes, example:
  mount -t weed fuse /mnt -o "filer='192.168.0.1:8888,192.168.0.2:8888',filer.path=/"

  To check valid options look "weed mount --help"
  `,
}
