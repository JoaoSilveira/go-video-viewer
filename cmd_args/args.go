package cmd_args

import "flag"

type CmdArgs struct {
	JsonFile string
}

func (args CmdArgs) HasJsonFile() bool {
	return args.JsonFile != ""
}

func ReadArgs() CmdArgs {
	args := CmdArgs{}
	flag.StringVar(&args.JsonFile, "json-file", "", "json file path")
	flag.Parse()

	return args
}
