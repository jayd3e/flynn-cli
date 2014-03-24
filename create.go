package main

import ()

var cmdCreate = &Command{
	Run:   runCreate,
	Usage: "create APP",
	Short: "create a Flynn app",
	Long:  `Creates a Flynn remote in your current git repository to push an app to`,
}

func runCreate(cmd *Command, args []string) {
	url, _ := gitRemoteUrlFromApp(args[0])
	addGitRemote("flynn", url)
}
